// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//! Writes requirements.json, the expectations of the Cargo requirements of the Versioner, to the
//! standard output, for a grid of requirements and versions. For each requirement it records
//! whether the semver crate 1.0.28, the parser and the matcher of Cargo, accepts it, and the
//! versions of the grid that it matches. For each requirement in the form that the semver crate
//! writes, it also records the requirement that cargo-release writes for each version of the grid,
//! by upgrade_requirement of cargo-release's src/ops/version.rs, where that requirement matches the
//! version.
//!
//! Run it from the directory of the module, with a target directory outside the repository:
//!
//! ```text
//! cargo run --quiet --manifest-path release/testdata/generate/Cargo.toml \
//!     --target-dir <dir> > release/testdata/requirements.json
//! ```

use std::fmt::Write as _;

use semver::{Comparator, Op, Version, VersionReq};

/// The versions of the grid, with pre-releases and build metadata around the bounds of the
/// requirements.
const VERSIONS: &[&str] = &[
    "0.0.0", "0.0.1", "0.0.2", "0.0.3", "0.0.4", "0.1.0", "0.2.0", "0.2.2", "0.2.3", "0.2.4", "0.3.0",
    "0.9.9", "1.0.0", "1.0.1", "1.1.0", "1.2.0", "1.2.2", "1.2.3", "1.2.4", "1.3.0", "1.9.9", "2.0.0",
    "2.0.1", "3.0.0", "1.2.3-alpha", "1.2.3-alpha.1", "1.2.3-alpha.2", "1.2.3-beta", "1.2.3-beta.2",
    "1.2.3-beta.3", "1.2.3-rc.1", "1.2.4-alpha", "1.3.0-alpha.1", "2.0.0-alpha", "2.0.0-0",
    "0.0.3-beta", "0.0.3-beta.1", "0.2.3-beta", "0.2.3-rc", "0.2.0-0", "0.1.0-rc.1", "1.2.3+build",
    "1.2.3-alpha.1+build", "0.0.0-0",
];

/// The operators of a comparator, with the empty operator of a requirement without one.
const OPERATORS: &[&str] = &["", "=", ">", ">=", "<", "<=", "~", "^"];

/// The versions of a comparator, of each precision, with pre-releases and build metadata.
const BODIES: &[&str] = &[
    "0", "0.0", "0.0.0", "0.0.3", "0.2", "0.2.3", "1", "1.2", "1.2.3", "2.0.0", "1.2.3-beta.2",
    "0.2.3-beta", "0.0.3-beta", "1.2.3-alpha", "1.2.3+build", "1.2.3-alpha.1+build",
];

/// The requirements of the grid beside the product of OPERATORS and BODIES: wildcards, more than
/// one comparator, spaces, and requirements that the semver crate refuses.
const REQUIREMENTS: &[&str] = &[
    "*", "x", "X", "1.*", "1.x", "1.X", "1.2.*", "1.*.*", "0.*", "0.0.*", "^1.*", "~1.2.*", ">=1.*",
    "=1.*", "<1.*", "<=1.2.*", ">1.*", "^0.*", "~0.*", ">=1.2, <2", ">=1.2,<2", ">= 1.2.0 , < 1.3",
    ">=0.1.0, <0.2.0-0", ">1.2.3-alpha, <1.2.3", "~1.2, ^1.2.5", "^1.2.3, ^1.2.3-beta.2",
    "=1.2.3-alpha.1, >1.0", ">=1.2.3-rc.1, <2", " ^1.2", "^ 1.2", "1.2 ", "  1.2.3  ", ">=  1.2",
    "^1.2 ,  <2", "18446744073709551615", "01.2", "1.02", "1.2.03", "^", ">", "a", "1.2.3.4",
    "1.*.3", "*, 1.0", "1.0, *", "1 2", "1.2.3-", "1.2.3+", "1.2.3-01", "~>1.2", "=>1.2",
    "1.2.3 - 2.0.0", "v1.2.3", "1.2.3-alpha..1", "   ", "\t1.2", "1.2,", ",1.2", "1.2 , ",
    "18446744073709551616", ">=1.2 <2", "1.2.3-beta+", "x.1", "*.1", "1.x.3", "^1.2.3 || ^2",
];

fn main() {
    let versions: Vec<Version> = VERSIONS.iter().map(|v| Version::parse(v).unwrap()).collect();
    let mut requirements: Vec<String> = Vec::new();
    for op in OPERATORS {
        for body in BODIES {
            requirements.push(format!("{op}{body}"));
        }
    }
    requirements.extend(REQUIREMENTS.iter().map(ToString::to_string));
    requirements.push(vec![">=0.0.0"; 32].join(", "));
    requirements.push(vec![">=0.0.0"; 33].join(", "));

    let mut out = String::from("{\n  \"generator\": \"semver 1.0.28, cargo-release upgrade_requirement\",\n");
    out.push_str("  \"versions\": [");
    for (i, v) in VERSIONS.iter().enumerate() {
        if i > 0 {
            out.push_str(", ");
        }
        out.push_str(&quote(v));
    }
    out.push_str("],\n  \"requirements\": [\n");
    for (i, text) in requirements.iter().enumerate() {
        let separator = if i + 1 < requirements.len() { "," } else { "" };
        let Ok(req) = VersionReq::parse(text) else {
            writeln!(out, "    {{\"requirement\": {}, \"valid\": false}}{separator}", quote(text)).unwrap();
            continue;
        };
        let matches: Vec<String> = VERSIONS
            .iter()
            .zip(&versions)
            .filter(|(_, v)| req.matches(v))
            .map(|(s, _)| quote(s))
            .collect();
        write!(
            out,
            "    {{\"requirement\": {}, \"valid\": true, \"matches\": [{}]",
            quote(text),
            matches.join(", ")
        )
        .unwrap();
        if canonical(text, &req) {
            let mut rewrites = Vec::new();
            for (s, v) in VERSIONS.iter().zip(&versions) {
                let Ok(upgraded) = upgrade_requirement(text, v) else {
                    continue;
                };
                let written = upgraded.unwrap_or_else(|| text.clone());
                if VersionReq::parse(&written).is_ok_and(|r| r.matches(v)) {
                    rewrites.push(format!("{}: {}", quote(s), quote(&written)));
                }
            }
            write!(out, ", \"rewrites\": {{{}}}", rewrites.join(", ")).unwrap();
        }
        writeln!(out, "}}{separator}").unwrap();
    }
    out.push_str("  ]\n}\n");
    print!("{out}");
}

/// Reports whether text is the requirement req as cargo-release writes it: the form of the semver
/// crate, without the caret of a requirement whose text has none.
fn canonical(text: &str, req: &VersionReq) -> bool {
    let mut written = req.to_string();
    if written.starts_with('^') && !text.starts_with('^') {
        written.remove(0);
    }
    written == text
}

/// Returns s as a string of JSON.
fn quote(s: &str) -> String {
    let mut out = String::from("\"");
    for c in s.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\t' => out.push_str("\\t"),
            '\n' => out.push_str("\\n"),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

/// upgrade_requirement of cargo-release's src/ops/version.rs, with the error of a String.
fn upgrade_requirement(req: &str, version: &Version) -> Result<Option<String>, String> {
    let req_text = req.to_owned();
    let raw_req = VersionReq::parse(&req_text).expect("semver to generate valid version requirements");
    if raw_req.comparators.is_empty() {
        // Empty matches everything, no-change.
        Ok(None)
    } else {
        let comparators: Result<Vec<_>, _> =
            raw_req.comparators.into_iter().map(|p| set_comparator(p, version)).collect();
        let comparators = comparators?;
        let new_req = VersionReq { comparators };
        let mut new_req_text = new_req.to_string();
        if new_req_text.starts_with('^') && !req.starts_with('^') {
            new_req_text.remove(0);
        }
        if new_req_text == req_text { Ok(None) } else { Ok(Some(new_req_text)) }
    }
}

/// set_comparator of cargo-release's src/ops/version.rs.
fn set_comparator(mut pred: Comparator, version: &Version) -> Result<Comparator, String> {
    match pred.op {
        Op::Wildcard => {
            pred.major = version.major;
            if pred.minor.is_some() {
                pred.minor = Some(version.minor);
            }
            if pred.patch.is_some() {
                pred.patch = Some(version.patch);
            }
            Ok(pred)
        }
        Op::Exact | Op::Tilde | Op::Caret => Ok(assign_partial_req(version, pred)),
        _ => Err(format!("support for modifying {pred} is currently unsupported")),
    }
}

/// assign_partial_req of cargo-release's src/ops/version.rs.
fn assign_partial_req(version: &Version, mut pred: Comparator) -> Comparator {
    pred.major = version.major;
    if pred.minor.is_some() {
        pred.minor = Some(version.minor);
    }
    if pred.patch.is_some() {
        pred.patch = Some(version.patch);
    }
    pred.pre = version.pre.clone();
    pred
}
