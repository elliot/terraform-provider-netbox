#!/usr/bin/env python3
"""Normalise the NetBox OpenAPI document before running openapi-generator.

This is a port of netbox-community/go-netbox's scripts/fix-spec.py extended for
NetBox 4.7 and for the needs of a Terraform provider. The goal is a generated Go
client whose surface is (a) stable across NetBox minor releases and (b)
mechanically predictable, so that internal/gen can emit calls against it from
the pristine spec.

Transformations (in order):

  1. Request bodies of the form ``oneOf: [X, array<X>]`` (bulk create) are
     collapsed to ``X``.
  2. Foreign keys written as ``oneOf: [integer, Brief*Request]`` are collapsed
     to ``type: integer`` (nullable preserved). Terraform always writes IDs.
     Likewise ``oneOf`` of two string branches (colour, dns_name, email, link)
     collapses to a plain string.
  3. Read-side choice objects ``{value: <enum>, label: <enum>}`` are hoisted to
     shared components ``ChoiceString`` / ``ChoiceInteger`` so the generated Go
     type name is deterministic.
  4. All remaining ``enum`` lists are removed: choice fields become plain strings
     or integers. The generated client therefore never rejects a value that a
     newer NetBox added, and the Terraform layer validates against the pristine
     spec instead.
  5. Read schemas (everything that is not a ``*Request``) lose their ``required``
     lists except ``id``: NetBox marks read-only counters and nullable fields as
     required, which makes the generated UnmarshalJSON reject legitimate
     responses.
  6. Request schemas: ``required`` is corrected for the known drf-spectacular
     quirks; ``NestedTagRequest`` only requires ``slug``; binary image fields
     are made non-nullable (openapi-generator issue #18006).
  7. ``VLANGroup.vid_ranges`` / ``IntegerRange`` are typed as ``[[int,int]]``.
  8. Non-2xx responses are dropped (they are ``oneOf`` unions of throw-away types).

Usage: fix_spec.py <input.json> <output.json>
"""

import copy
import json
import sys

CHOICE_STRING = "ChoiceString"
CHOICE_INTEGER = "ChoiceInteger"

# Request schemas whose ``required`` list disagrees with NetBox's real validation.
REQUIRED_FIXES = {
    # schema: (add, remove)
    "WritableRackTypeRequest": ([], ["form_factor"]),  # deprecated in 4.7, has a default
    "CustomFieldRequest": ([], ["type"]),
    "IKEPolicyRequest": ([], ["version"]),
    "TunnelRequest": ([], ["status"]),
    "TunnelTerminationRequest": ([], ["role"]),
    "NestedTagRequest": (["slug"], ["name"]),
}

BINARY_FIELDS = {"front_image", "rear_image", "image"}


def ref_name(ref):
    return ref.rsplit("/", 1)[-1]


def is_request_schema(name):
    return name.endswith("Request")


def collapse_bulk_bodies(paths):
    n = 0
    for path, item in paths.items():
        for method, op in item.items():
            if not isinstance(op, dict):
                continue
            body = op.get("requestBody", {}).get("content", {})
            for ctype, media in body.items():
                schema = media.get("schema", {})
                one = schema.get("oneOf")
                if not one or len(one) != 2:
                    continue
                objs = [s for s in one if "$ref" in s]
                arrs = [s for s in one if s.get("type") == "array"]
                if len(objs) == 1 and len(arrs) == 1:
                    media["schema"] = objs[0]
                    n += 1
    return n


def drop_error_responses(paths):
    """Remove non-2xx responses: NetBox declares them as ``oneOf`` unions which
    only produce throw-away Go types. Error bodies are parsed by hand."""
    n = 0
    for path, item in paths.items():
        for method, op in item.items():
            if not isinstance(op, dict):
                continue
            for code in list(op.get("responses", {}).keys()):
                if not str(code).startswith("2"):
                    del op["responses"][code]
                    n += 1
    return n


def collapse_fk_oneof(schemas):
    """oneOf [integer, allOf[$ref Brief*Request]] -> integer."""
    n = 0
    for name, schema in schemas.items():
        for pname, prop in schema.get("properties", {}).items():
            one = prop.get("oneOf")
            if not one:
                continue
            ints = [s for s in one if s.get("type") == "integer"]
            refs = [s for s in one if "$ref" in s or "allOf" in s]
            if len(ints) == 1 and len(refs) == 1 and len(one) == 2:
                new = {"type": "integer"}
                if prop.get("nullable") or any(s.get("nullable") for s in one):
                    new["nullable"] = True
                for k in ("description", "title"):
                    if k in prop:
                        new[k] = prop[k]
                schema["properties"][pname] = new
                n += 1
    return n


def collapse_string_oneof(schemas):
    """oneOf [string(pattern), string(maxLength 0)] (colors, dns_name, email, link) -> string."""
    n = 0
    for name, schema in schemas.items():
        for pname, prop in schema.get("properties", {}).items():
            one = prop.get("oneOf")
            if not one or not all(s.get("type") == "string" for s in one):
                continue
            new = {"type": "string"}
            first = one[0]
            for k in ("maxLength", "pattern", "format"):
                if k in first:
                    new[k] = first[k]
            for k in ("description", "title", "nullable"):
                if k in prop:
                    new[k] = prop[k]
            schema["properties"][pname] = new
            n += 1
    return n


def hoist_choice_objects(schemas):
    """Replace inline {value,label} objects by $ref to ChoiceString/ChoiceInteger."""
    schemas[CHOICE_STRING] = {
        "type": "object",
        "description": "A NetBox choice field as returned on read: machine value plus human label.",
        "properties": {
            "value": {"type": "string", "nullable": True},
            "label": {"type": "string", "nullable": True},
        },
    }
    schemas[CHOICE_INTEGER] = {
        "type": "object",
        "description": "A NetBox integer choice field as returned on read: machine value plus human label.",
        "properties": {
            "value": {"type": "integer", "nullable": True},
            "label": {"type": "string", "nullable": True},
        },
    }
    n = 0
    for name, schema in list(schemas.items()):
        if name in (CHOICE_STRING, CHOICE_INTEGER):
            continue
        for pname, prop in schema.get("properties", {}).items():
            if prop.get("type") != "object":
                continue
            props = prop.get("properties") or {}
            if set(props.keys()) != {"value", "label"}:
                continue
            vtype = props["value"].get("type")
            if vtype is None and "enum" in props["value"]:
                # e.g. DataSource.type value has enum with null and no type
                vtype = "string"
            target = CHOICE_INTEGER if vtype == "integer" else CHOICE_STRING
            new = {"allOf": [{"$ref": f"#/components/schemas/{target}"}]}
            if prop.get("nullable"):
                new["nullable"] = True
            if prop.get("readOnly"):
                new["readOnly"] = True
            if "description" in prop:
                new["description"] = prop["description"]
            schema["properties"][pname] = new
            n += 1
    return n


def strip_enums(node):
    """Recursively drop every enum list (and x-spec-enum-id)."""
    n = 0
    if isinstance(node, dict):
        if "enum" in node:
            del node["enum"]
            n += 1
            # A property that was only an enum still needs a type.
            if "type" not in node:
                node["type"] = "string"
        node.pop("x-spec-enum-id", None)
        for v in list(node.values()):
            n += strip_enums(v)
    elif isinstance(node, list):
        for v in node:
            n += strip_enums(v)
    return n


def relax_read_required(schemas):
    n = 0
    for name, schema in schemas.items():
        if is_request_schema(name):
            continue
        req = schema.get("required")
        if not req:
            continue
        keep = [r for r in req if r == "id"]
        if keep:
            schema["required"] = keep
        else:
            schema.pop("required", None)
        n += 1
    return n


def fix_request_required(schemas):
    n = 0
    for name, (add, remove) in REQUIRED_FIXES.items():
        schema = schemas.get(name)
        if not schema:
            continue
        req = list(schema.get("required") or [])
        req = [r for r in req if r not in remove]
        for a in add:
            if a not in req:
                req.append(a)
        if req:
            schema["required"] = req
        else:
            schema.pop("required", None)
        n += 1
    # Patched* variants never have required lists; nothing to do.
    return n


def fix_binary_and_ranges(schemas):
    n = 0
    for name, schema in schemas.items():
        for pname, prop in schema.get("properties", {}).items():
            if pname in BINARY_FIELDS and prop.get("format") == "binary":
                prop.pop("nullable", None)
                n += 1
    for name in ("IntegerRange", "IntegerRangeRequest"):
        if name in schemas:
            schemas[name] = {
                "type": "array",
                "items": {"type": "integer", "format": "int32"},
                "minItems": 2,
                "maxItems": 2,
                "description": "Inclusive [start, end] range.",
            }
            n += 1
    return n


def drop_unused_plain_request_twins(schemas, paths):
    """No-op placeholder: plain *Request twins of Writable*Request are kept; they
    are referenced by bulk delete bodies and are harmless."""
    return 0


def main(src, dst):
    with open(src) as f:
        doc = json.load(f)
    schemas = doc["components"]["schemas"]
    paths = doc["paths"]
    stats = {}
    stats["bulk_bodies"] = collapse_bulk_bodies(paths)
    stats["error_responses"] = drop_error_responses(paths)
    stats["fk_oneof"] = collapse_fk_oneof(schemas)
    stats["string_oneof"] = collapse_string_oneof(schemas)
    stats["choice_objects"] = hoist_choice_objects(schemas)
    stats["read_required"] = relax_read_required(schemas)
    stats["request_required"] = fix_request_required(schemas)
    stats["binary_ranges"] = fix_binary_and_ranges(schemas)
    stats["enums"] = strip_enums(doc)
    doc.setdefault("info", {})["x-fix-spec"] = stats
    with open(dst, "w") as f:
        json.dump(doc, f, indent=1, sort_keys=False)
    print(json.dumps(stats))


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    main(sys.argv[1], sys.argv[2])
