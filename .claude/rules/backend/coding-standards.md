---
paths: pkg/**/*.go
---

# Coding Standards

## CRITICAL - No TODO or Future Comments
**NEVER leave TODO, FIXME, XXX, HACK, or "in future" comments in code unless explicitly instructed by the user.**

### Rules
- **Implement features completely** - Do not add placeholder comments for future work
- **If a feature cannot be implemented now**, ask the user for clarification instead of leaving a TODO
- **No "Note: will be used in future"** comments - Either implement it now or don't add it at all
- **No deferring implementation** with comments like "this will be added later"
- **Complete all integration work** - Don't create infrastructure that isn't used immediately

### Examples of PROHIBITED patterns
```go
// BAD - TODO comment without user instruction
// TODO: Add validation here

// BAD - Future placeholder
// Note: This will be used in future when X supports Y

// BAD - Deferred implementation
// This feature will be implemented later

// BAD - Unused variable with future comment
firestoreRepo := createRepo()
// Note: Firestore repository is created but not yet integrated
_ = firestoreRepo
```

### GOOD pattern - Complete implementation
```go
// GOOD - Feature is fully implemented
if firestoreConfig.Enabled() {
    repo := createRepo()
    defer repo.Close()
    clients = append(clients, infra.WithScanRepository(repo))
}
```
