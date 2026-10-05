---
adr: 0008
title: The npm and Gradle toolchains are in the JavaScript and Java modules
status: Accepted
date: 2026-10-05
supersedes: ADR-0001 and ADR-0007, in part
superseded-by: none
rfc: RFC-0001
---

# ADR-0008: The npm and Gradle toolchains are in the JavaScript and Java modules

## Status

Accepted

## Context

ADR-0001 put the npm and Gradle toolchains in `ergon-lang`, as the packages `ergon-lang/js` and `ergon-lang/jvm`, and ADR-0007 registers them from there. Since ADR-0007, a toolchain is a catalog entry of its own, and seven languages declare their toolchain in their own module. A module per language exists so that a module's tests need one toolchain. The release roles of the npm and Gradle toolchains run npm, pnpm, bun and Gradle, so their packages in `ergon-lang` would make that module's tests need Node and a JDK with Gradle.

## Decision

We will declare the js toolchain in the JavaScript module and the jvm toolchain in the Java module, because every toolchain is then in a language module and the tests of each module need one toolchain.

`ergon-lang-javascript` declares JavaScript and the toolchain `js`. `ergon-lang-java` declares Java and the toolchain `jvm`. `ergon-lang-typescript` imports the root package of `ergon-lang-javascript`, and `ergon-lang-kotlin` imports the root package of `ergon-lang-java`, for the name of their toolchain. JavaScript is a language of ergon, so ergon supports eleven languages. `ergon-lang` contains only machinery that two or more languages use.

## Alternatives Considered

### Packages in ergon-lang

`ergon-lang/js` and `ergon-lang/jvm`, as ADR-0001 decided. It lost because the tests of `ergon-lang` would need two toolchains, and because the npm toolchain served one language of ergon.

### A module for each toolchain

A module of its own for each toolchain: `ergon-lang-js` and `ergon-lang-jvm`. It lost because a module that declares a toolchain without a language would be a third kind of module, with rules of its own. The seven other toolchains are in the module of their language.

## Consequences

**Positive:**

- Every toolchain is in a language module, and every language module has one toolchain to test against.
- `ergon-lang` contains no toolchain.

**Negative:**

- Two language modules import another language module, as two exceptions to the rule that keeps language modules apart. depguard allows each by its exact path.
- A change to the root package of `ergon-lang-java` or `ergon-lang-javascript` can change the build of Kotlin or TypeScript.

**Neutral:**

- ergon has eleven language modules and fifteen modules in all.
