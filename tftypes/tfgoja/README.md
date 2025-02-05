# tfgoja

This package converts a Terraform value into a goja value using reflection techniques and JSON.

It can be considered a fork of [go-cty-goja](https://github.com/zclconf/go-cty-goja), but adapted to work with the Terraform provider SDK framework.

This package tries to convert values to JSON and then from JSON into either Terraform or goja compatible values. Because of this, a full round-trip from Terraform to goja and back (or the other way around) is lossy.
Known limitations:
- It doesn't support sets: there are no sets in JSON - only arrays. They are seen as a arrays and converted stored internally as slices.
- It doesn't know to make a difference between arrays and tuples, so they will be generalized as tuples.
- It doesn't know to make a difference between maps and objects, so maps will be converted into objects.

Depending on how this payload is manipulated, type alterations can happen outside of this package.
