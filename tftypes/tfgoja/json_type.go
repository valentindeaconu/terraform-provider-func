package tfgoja

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	NullType attr.Type = basetypes.NewDynamicNull().Type(context.Background())
)

// ImpliedType returns the attr Type implied by the structure of the given
// JSON-compliant buffer. This function implements the default type mapping
// behavior used when decoding arbitrary JSON without explicit Terraform Type
// information.
//
// The rules are as follows:
//
// JSON strings, numbers and bools map to their equivalent primitive type in
// Terraform.
//
// JSON objects map to Terraform object types, with the attributes defined by
// the object keys and the types of their values.
//
// JSON arrays map to Terraform tuple types, with the elements defined by the
// types of the array members.
//
// Any nulls are typed as DynamicPseudoType, so callers of this function
// must be prepared to deal with this. Callers that do not wish to deal with
// dynamic typing should not use this function and should instead describe
// their required types explicitly with a attr.Type instance when decoding.
//
// Any JSON syntax errors will be returned as an error, and the type will
// be the invalid value NullType.
func JSONImpliedType(buf []byte) (attr.Type, error) {
	r := bytes.NewReader(buf)
	dec := json.NewDecoder(r)
	dec.UseNumber()

	ty, err := impliedType(dec)
	if err != nil {
		return NullType, err
	}

	if dec.More() {
		return NullType, fmt.Errorf("extraneous data after JSON object")
	}

	return ty, nil
}

func impliedType(dec *json.Decoder) (attr.Type, error) {
	tok, err := dec.Token()
	if err != nil {
		return NullType, err
	}

	return impliedTypeForTok(tok, dec)
}

func impliedTypeForTok(tok json.Token, dec *json.Decoder) (attr.Type, error) {
	if tok == nil {
		return NullType, nil
	}

	switch ttok := tok.(type) {
	case bool:
		return basetypes.BoolType{}, nil

	case json.Number:
		return basetypes.NumberType{}, nil

	case string:
		return basetypes.StringType{}, nil

	case json.Delim:

		switch rune(ttok) {
		case '{':
			return impliedObjectType(dec)
		case '[':
			return impliedTupleType(dec)
		default:
			return NullType, fmt.Errorf("unexpected token %q", ttok)
		}

	default:
		return NullType, fmt.Errorf("unsupported JSON token %#v", tok)
	}
}

func impliedObjectType(dec *json.Decoder) (attr.Type, error) {
	// By the time we get in here, we've already consumed the { delimiter
	// and so our next token should be the first object key.

	var atys map[string]attr.Type

	for {
		// Read the object key first
		tok, err := dec.Token()
		if err != nil {
			return NullType, err
		}

		if ttok, ok := tok.(json.Delim); ok {
			if rune(ttok) != '}' {
				return NullType, fmt.Errorf("unexpected delimiter %q", ttok)
			}
			break
		}

		key, ok := tok.(string)
		if !ok {
			return NullType, fmt.Errorf("expected string but found %T", tok)
		}

		// Now read the value
		tok, err = dec.Token()
		if err != nil {
			return NullType, err
		}

		aty, err := impliedTypeForTok(tok, dec)
		if err != nil {
			return NullType, err
		}

		if atys == nil {
			atys = make(map[string]attr.Type)
		}

		atys[key] = aty
	}

	return basetypes.ObjectType{AttrTypes: atys}, nil
}

func impliedTupleType(dec *json.Decoder) (attr.Type, error) {
	// By the time we get in here, we've already consumed the [ delimiter
	// and so our next token should be the first value.

	var etys []attr.Type

	for {
		tok, err := dec.Token()
		if err != nil {
			return NullType, err
		}

		if ttok, ok := tok.(json.Delim); ok {
			if rune(ttok) == ']' {
				break
			}
		}

		ety, err := impliedTypeForTok(tok, dec)
		if err != nil {
			return NullType, err
		}
		etys = append(etys, ety)
	}

	return basetypes.TupleType{ElemTypes: etys}, nil
}
