# Adversarial Review: Admin Dashboard Revival Design

**Phase:** design
**Artifact:** `docs/plans/2026-05-26-admin-dashboard-revival-design.md`
**Status:** PASS after R1 revisions

## Findings

| sev | class | loc | issue | fix |
|---|---|---|---|---|
| Important | Security / repo precedent | Architecture | R1 design implied admin subprocess could call authz plugin directly. External plugin precedent exposes host-to-plugin calls, not arbitrary plugin-to-plugin service invocation. | Revised: auth/authz composed through Workflow routes/pipelines; admin consumes authz evidence and denies absent evidence. |
| Important | YAGNI / ownership | Infrastructure | R1 design mentioned persistent contribution registry, risking duplicated declarative config and plugin-owned state. | Revised: contributions are runtime/declarative registrations re-applied at startup; domain state stays with owning plugin/app. |
| Minor | User-intent drift | Rendering | R1 design said asset/iframe modes may render as links first, which risks under-delivering rich plugin surfaces. | Plan must implement usable internal/json-schema shell and treat asset/iframe contracts as mounted navigable surfaces, not hidden future work. |

## Bug-Class Scan Transcript

| class | result | note |
|---|---|---|
| Project-guidance conflicts | Clean | Revised design follows workspace guidance: Workflow dogfooding, plugin boundaries, strict contracts, scenario proof. |
| Assumptions under attack | Finding resolved | A1/A2 were fragile; revised to avoid plugin-to-plugin direct invocation and require explicit auth/authz primitive tasks if gaps appear. |
| Repo-precedent conflicts | Finding resolved | Existing SDK supports plugin-provided modules/steps/service invocation from host; design now composes authz in host workflows. |
| YAGNI violations | Finding resolved | Removed admin-owned persistence; retained rendering modes because user explicitly asked for module-provided meaningful admin UI. |
| Missing failure modes | Clean | Authz misconfiguration, missing authz evidence, malformed contribution, and startup re-registration are covered. |
| Security / privacy | Finding resolved | Direct authz call assumption fixed; default deny and route/pipeline enforcement are explicit. |
| Infrastructure impact | Clean | Optional admin route/port only; no new cloud/storage/secret requirement. |
| Multi-component validation | Clean | Design requires plugin tests plus scenario across primary app, admin, auth, authz, and contributed surface. |
| Rollback story | Clean | Plugin, scenario, UI, and route rollback paths stated. |
| Simpler alternative | Clean | Static dashboard rejected because it cannot manage contributed actions or auth/authz. |
| User-intent drift | Clean | Scope retains mini-app, module UI registration, auth/authz management, strict contracts, and scenarios. |

## Options Considered

1. Host-level UI manifest only: simpler, but lacks typed admin resource/action contracts and authz evidence path.
2. Separate admin sidecar only: stronger isolation, but makes app awareness and token/permission sharing harder; retained as a mode, not the default.

## Verdict Reasoning

R1 found two tangible Important issues and one Minor delivery risk. The design
was revised to remove the unsafe plugin-to-plugin authz assumption and avoid
unneeded admin persistence. Remaining concerns are implementation discipline,
not blocking design flaws.

