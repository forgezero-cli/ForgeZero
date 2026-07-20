# ForgeZero package manager

ForgeZero includes a lightweight package-management layer that can verify and install local packages after they have been signed and trusted.

## Trust model

The package manager is intentionally conservative. A package is only accepted if:

- it includes a valid signature block,
- the signing key is already trusted,
- the package manifest passes verification.

This makes installation predictable and reduces the risk of accepting untrusted artifacts.

## Typical commands

- `fz pm trust <key>`: add a key to the trusted-key store.
- `fz pm keys`: list trusted keys.
- `fz pm verify <package>`: verify a package manifest.
- `fz pm install <package>`: install a package after verification.

## Why this exists

The package manager is not meant to replace a full package ecosystem. It is a verification-oriented mechanism for controlled, auditable installs in a build-oriented environment.
