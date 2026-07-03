# Supply Chain Security: SBOM & SAST Audit

## SBOM generation (`fz sbom`)

Generates a CycloneDX SBOM (JSON) for build components.

```bash
fz sbom
fz sbom -o /tmp/myproject-sbom.cdx.json
fz sbom -dir ./src -o release-sbom.cdx.json
```

SBOM is written atomically to avoid partial output.

---

## SAST audit scanner (`fz audit`)

```bash
fz audit
fz audit -dir ./src
fz audit -json
```

### Checks

1) Hardcoded secrets
- Detects high-entropy patterns and common secret assignments.
- Findings include file/line/severity; it does not print the secret value.

2) License compliance
- Flags strong copyleft and proprietary vendored code with severity levels.

3) Dangerous patterns
- Detects unsafe functions (e.g. `gets`, dangerous `sprintf` usage), unchecked allocations, and risky shell patterns.

### Exit codes

- `0` = no findings
- `1` = any WARNING or above

