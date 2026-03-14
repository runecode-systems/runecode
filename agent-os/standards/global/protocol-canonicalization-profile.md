# Protocol Canonicalization Profile

Use the MVP JSON canonicalization/signing profile for hashed or signed protocol objects.

- Sign the detached payload's RFC 8785 JCS bytes, not wrapper fields or language-native serialization
- Keep canonicalized numbers within the shared Go/JS safe-integer range
- Reject decimal and exponent numeric lexemes fail-closed before hashing
- Normalize `-0` to `0`
- Reject non-ASCII object keys in the MVP profile
- Prove parity with checked-in golden canonical JSON and hash fixtures
- No exceptions until a later spec widens the profile
