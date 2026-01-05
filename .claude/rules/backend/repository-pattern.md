---
paths: pkg/repository/**/*.go
---

# Repository Pattern

## Dual Implementation Strategy
All repositories must have both Memory and Firestore implementations:
- **Memory**: In-memory implementation for testing and development ([pkg/repository/memory/](pkg/repository/memory/))
- **Firestore**: Production implementation using Firestore ([pkg/repository/firestore/](pkg/repository/firestore/))

## Firestore Collection Structure
```
repo/{owner:repo}/branch/{name}/target/{id}/vulnerability/{id}
```

Document ID format uses `:` separator (e.g., `owner:repo`) since GitHub names cannot contain colons.

## Interface Definition
All repository interfaces are defined in [pkg/domain/interfaces/](pkg/domain/interfaces/).

## Testing Requirements
See [backend/testing.md](backend/testing.md) for complete testing patterns including:
- Common test helper approach
- Environment-based Firestore testing
- Test ID randomization with UUID
