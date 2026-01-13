# openapi-ts-enumgen

Generate TypeScript enum-like constants from OpenAPI schemas.

This tool generates enum-like const objects, helper lists, and utility functions  
from enum definitions in OpenAPI components/schemas.

It is designed to complement OpenAPI generators that already output  
TypeScript union types, without introducing additional runtime dependencies.

## Installation

### Using go install

```bash
go install github.com/taknb2nch/openapi-ts-enumgen@latest
```

Make sure $GOPATH/bin (or $GOBIN) is in your PATH.

## Usage

```bash
openapi-ts-enumgen --input openapi.yaml --output enums.ts
```

### Options

--input  
Path to the OpenAPI YAML file.

--output  
Path to the generated TypeScript file.

--quote  
Quote style for string literals: single or double (default: double).

--no-sort  
Disable sorting of schema names in the output.  
Enum item order always follows the OpenAPI definition.

### Example

```bash
openapi-ts-enumgen --input openapi.yaml --output src/enums.ts --quote single
```

## OpenAPI Example

Example enum definition in OpenAPI (YAML):

```yaml
components:
  schemas:
    StatusCode:
      type: string
      description: ステータスコード
      enum:
        - active   # 有効
        - inactive # 無効
        - archived # アーカイブ済み
```

## Generated Output (Example)

TypeScript code generated from the schema above:

```ts
/**
 * ステータスコード
 *
 * Defined values:
 * - `StatusCode.Active`: 有効
 * - `StatusCode.Inactive`: 無効
 * - `StatusCode.Archived`: アーカイブ済み
 *
 * @see OpenAPI components/schemas/StatusCode (openapi.yaml)
 */
export const StatusCode = {
  Active: 'active',
  Inactive: 'inactive',
  Archived: 'archived',
} as const;

/**
 * StatusCode value type (enum values).
 */
export type StatusCodeEnum =
  (typeof StatusCode)[keyof typeof StatusCode];

/**
 * ステータスコード (in OpenAPI definition order)
 */
export const StatusCodeList = [
  StatusCode.Active,
  StatusCode.Inactive,
  StatusCode.Archived,
] as const;

/**
 * ステータスコード (select options)
 */
export const StatusCodeOptions = [
  { label: '有効', value: StatusCode.Active },
  { label: '無効', value: StatusCode.Inactive },
  { label: 'アーカイブ済み', value: StatusCode.Archived },
] as const;

/**
 * ステータスコード (value → label)
 */
export const StatusCodeLabel = {
  [StatusCode.Active]: '有効',
  [StatusCode.Inactive]: '無効',
  [StatusCode.Archived]: 'アーカイブ済み',
} as const;

/**
 * Check whether a value is a valid StatusCode.
 */
export function isStatusCode(
  value: unknown
): value is StatusCodeEnum {
  return StatusCodeList.includes(value as StatusCodeEnum);
}

/**
 * Try to parse a value as StatusCode.
 * Returns undefined if the value is invalid.
 */
export function tryParseStatusCode(
  value: unknown
): StatusCodeEnum | undefined {
  return isStatusCode(value) ? value : undefined;
}
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
