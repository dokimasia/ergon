// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Writes satisfies.json and rewrites.json, the expectations of the ranges of the Versioner, into the
// directory out, for a grid of ranges and versions. satisfies.json has the answer of satisfies of
// node-semver 7.8.5 with its default options for each range and version. rewrites.json has the
// requirement that changesets writes for each requirement and new version, by its rules in
// apply-release-plan/src/version-package.ts, and null where changesets leaves the requirement as it
// is.
//
// Run it with the directory of an installed node-semver 7.8.5:
//
//	node generate.mjs <semver dir> <out>
import { createRequire } from "node:module";
import { writeFileSync } from "node:fs";
import { join } from "node:path";

const require = createRequire(import.meta.url);
const semver = require(process.argv[2]);
const out = process.argv[3];

const ranges = [
  "1.2.3", "=1.2.3", "v1.2.3", ">1.2.3", ">=1.2.3", "<1.2.3", "<=1.2.3", ">1.2", "<=1.2", ">=1", "<1",
  ">1.2.3-beta.1", ">=1.2.3-beta.2 <1.3.0", "~1.2.3", "~1.2", "~1", "~0.2.3", "~0.2", "~0", "~1.2.3-beta.2",
  "~>1.2.3", "~*", "^1.2.3", "^0.2.3", "^0.0.3", "^1.2.3-beta.2", "^0.2.3-beta", "^0.0.3-beta", "^1.2.x",
  "^0.0.x", "^0.0", "^1.x", "^0.x", "^1", "^0", "^x", "*", "", "x", "1.x", "1.2.x", "1.X", "1.*", "1", "1.2",
  "=1.x", "x.1", "1.x.3", "1.2.3 - 2.3.4", "1.2 - 2.3.4", "1 - 2.3.4", "* - 2.3.4", "1.2.3 - 2.3",
  "1.2.3 - 2", "1.2.3 - *", "1.2.3 - 2.3.4-beta", "1.2.3-beta - 2.0.0", "1.2.7 || >=1.2.9 <2.0.0",
  "^1.0.0 || ^2.0.0", ">=1.0.0 <1.5.0 || >=2.0.0", "<0.0.0-0 || ^1.0.0", "<0.0.0-0 || >*", "^1.0.0 || *",
  "^1.0.0 <0.0.0-0", ">=1.2.3 >=1.2.3", ">= 1.2.3", ">=1.2.3   <2", "  ^1.2.3  ", "^1.2.3+build", ">=0.0.0",
  "<0.0.0-0", ">*", "<*", ">=*", "<=1.x", ">1.x", "<1.x", ">=1.2.3 <=1.2.3", "1.0.0 >=0.0.0 <2",
  "not-a-range", "1.2.3.4", ">>1", "latest", "file:../a", "npm:other@^1.0.0", "github:owner/repo",
  "^01.2.3", "1.2.3 2.0.0", "^9007199254740991.0.0", "^9007199254740993.0.0", "^99999999999999999999.0.0",
];

const versions = [
  "0.0.0", "0.0.3", "0.0.4", "0.1.0", "0.2.3", "0.2.4", "0.3.0", "1.0.0", "1.2.2", "1.2.3", "1.2.4", "1.2.7",
  "1.2.8", "1.2.9", "1.3.0", "1.5.0", "2.0.0", "2.3.4", "2.3.5", "2.4.0", "3.0.0", "1.2.3-beta.1",
  "1.2.3-beta.2", "1.2.3-beta.3", "1.2.4-alpha", "1.3.0-0", "2.0.0-0", "0.0.3-beta", "0.0.3-rc.1",
  "0.2.3-beta", "0.2.3-gamma", "2.3.4-beta", "2.3.4-alpha", "1.2.3+build.5",
];

const satisfies = [];
for (const range of ranges) {
  for (const version of versions) {
    satisfies.push({ range, version, want: semver.satisfies(version, range) });
  }
}

// The rewrite of changesets for a range of a dependency and the new version of the dependency, with
// the release type that gives the new version.
function newRange(versionRange, newVersion, releaseType) {
  const comparatorSets = new semver.Range(versionRange).set;
  const comparators = comparatorSets.length === 1 ? comparatorSets[0] : [];
  if (comparators.length === 2 && versionRange.includes(">") && versionRange.includes("<")) {
    const lowerBound = comparators.find(({ operator }) => operator.startsWith(">"));
    const upperBound = comparators.find(({ operator }) => operator.startsWith("<"));
    if (lowerBound && upperBound) {
      let newUpperBound = upperBound.value;
      if (!semver.satisfies(newVersion, newUpperBound)) {
        newUpperBound = `<${semver.inc(newVersion, releaseType)}`;
      }
      return comparators[0] === lowerBound ? `>=${newVersion} ${newUpperBound}` : `${newUpperBound} >=${newVersion}`;
    }
  }
  let type = "";
  if (versionRange.charAt(0) === "^") type = "^";
  else if (versionRange.charAt(0) === "~") type = "~";
  else if (versionRange.startsWith(">=")) type = ">=";
  else if (versionRange.startsWith("<=")) type = "<=";
  else if (versionRange.charAt(0) === ">") type = ">";
  return `${type}${newVersion}`;
}

// The requirement that changesets writes for req and the new version, and null where it leaves req
// as it is.
function rewrite(req, newVersion, releaseType) {
  if (req.startsWith("file:") || req.startsWith("link:")) return null;
  let range = req;
  const workspace = req.startsWith("workspace:");
  if (workspace) {
    range = req.replace(/^workspace:/, "");
    if (range === "*" || range === "^" || range === "~" || semver.validRange(range) == null) return null;
  } else if (semver.validRange(range) == null) {
    return null;
  }
  if (new semver.Range(range).range === "" && semver.prerelease(newVersion) == null) return null;
  const rewritten = newRange(range, newVersion, releaseType);
  return workspace ? `workspace:${rewritten}` : rewritten;
}

const rewrites = [];
const reqs = [
  "^1.2.3", "~1.2.3", ">=1.2.3", "<=1.2.3", ">1.2.3", "1.2.3", "=1.2.3", "^0.2.3", ">=1.0.0 <2.0.0",
  "<2.0.0 >=1.0.0", ">=1.0.0 <1.3.0", "1.0.0 >=0.0.0 <2", "^1.0.0 || ^2.0.0", "*", "x", "", "1.x",
  "workspace:^1.2.3", "workspace:~1.2.3", "workspace:1.2.3", "workspace:>=1.0.0 <2.0.0", "workspace:*",
  "workspace:^", "workspace:~", "workspace:../a", "file:../a", "link:../a", "latest", "npm:other@^1.0.0",
];
const releases = [
  ["1.2.4", "patch"], ["1.3.0", "minor"], ["2.0.0", "major"], ["2.1.0", "minor"], ["2.0.1", "patch"],
  ["1.3.0-beta.0", "minor"],
];
for (const req of reqs) {
  for (const [version, type] of releases) {
    rewrites.push({ req, version, want: rewrite(req, version, type) });
  }
}

writeFileSync(join(out, "satisfies.json"), JSON.stringify(satisfies, null, 1) + "\n");
writeFileSync(join(out, "rewrites.json"), JSON.stringify(rewrites, null, 1) + "\n");
process.stdout.write(`${satisfies.length} satisfies, ${rewrites.length} rewrites\n`);
