# Roadmap And Major Release Direction

Hookr is moving to one primary public model:

- FlatBuffers-defined contracts
- generated host SDK and plugin PDK
- method-ID ABI runtime path

## Why A Major Version

The old mixed API story (string operations, multiple serialization pathways) is
being removed in favor of a single schema-driven surface.

This is a deliberate breaking simplification:

- clearer developer experience
- lower maintenance complexity
- stronger correctness guarantees

## Current Roadmap Themes

- harden runtime safety under hostile/malformed plugins
- complete generator consistency around FlatBuffers-first behavior
- keep Go path excellent, then expand language backends
- maintain benchmark discipline for hot-loop workloads

Further detail:

- [Major Release Migration Note](../major-release.md)
- [Implementation Plan](../implementation-plan.md)
