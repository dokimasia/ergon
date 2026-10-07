// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package spdx

import (
	"iter"
	"slices"
)

// ID is the SPDX identifier of a license that ergon supports, such as "MIT". The zero value is not
// a valid identifier.
type ID string

// The identifiers of the licenses that ergon supports, in the order of the SPDX License List. The
// docblock of each states the license's name in the SPDX License List 3.29.0.
const (
	// ZeroBSD is the BSD Zero Clause License.
	ZeroBSD ID = "0BSD"

	// AFL30 is the Academic Free License v3.0.
	AFL30 ID = "AFL-3.0"

	// AGPL30Only is the GNU Affero General Public License v3.0 only.
	AGPL30Only ID = "AGPL-3.0-only"

	// AGPL30OrLater is the GNU Affero General Public License v3.0 or later.
	AGPL30OrLater ID = "AGPL-3.0-or-later"

	// Apache20 is the Apache License 2.0.
	Apache20 ID = "Apache-2.0"

	// Artistic20 is the Artistic License 2.0.
	Artistic20 ID = "Artistic-2.0"

	// BlueOak100 is the Blue Oak Model License 1.0.0.
	BlueOak100 ID = "BlueOak-1.0.0"

	// BSD2Clause is the BSD 2-Clause "Simplified" License.
	BSD2Clause ID = "BSD-2-Clause"

	// BSD2ClausePatent is the BSD-2-Clause Plus Patent License.
	BSD2ClausePatent ID = "BSD-2-Clause-Patent"

	// BSD3Clause is the BSD 3-Clause "New" or "Revised" License.
	BSD3Clause ID = "BSD-3-Clause"

	// BSD3ClauseClear is the BSD 3-Clause Clear License.
	BSD3ClauseClear ID = "BSD-3-Clause-Clear"

	// BSD4Clause is the BSD 4-Clause "Original" or "Old" License.
	BSD4Clause ID = "BSD-4-Clause"

	// BSL10 is the Boost Software License 1.0.
	BSL10 ID = "BSL-1.0"

	// BUSL11 is the Business Source License 1.1. Its text states parameters, such as the change
	// date, before its terms.
	BUSL11 ID = "BUSL-1.1"

	// CC010 is the Creative Commons Zero v1.0 Universal.
	CC010 ID = "CC0-1.0"

	// CECILL21 is the CeCILL Free Software License Agreement v2.1.
	CECILL21 ID = "CECILL-2.1"

	// ECL20 is the Educational Community License v2.0.
	ECL20 ID = "ECL-2.0"

	// EPL10 is the Eclipse Public License 1.0.
	EPL10 ID = "EPL-1.0"

	// EPL20 is the Eclipse Public License 2.0.
	EPL20 ID = "EPL-2.0"

	// EUPL11 is the European Union Public License 1.1.
	EUPL11 ID = "EUPL-1.1"

	// EUPL12 is the European Union Public License 1.2.
	EUPL12 ID = "EUPL-1.2"

	// GPL20Only is the GNU General Public License v2.0 only.
	GPL20Only ID = "GPL-2.0-only"

	// GPL20OrLater is the GNU General Public License v2.0 or later.
	GPL20OrLater ID = "GPL-2.0-or-later"

	// GPL30Only is the GNU General Public License v3.0 only.
	GPL30Only ID = "GPL-3.0-only"

	// GPL30OrLater is the GNU General Public License v3.0 or later.
	GPL30OrLater ID = "GPL-3.0-or-later"

	// ISC is the ISC License.
	ISC ID = "ISC"

	// LGPL21Only is the GNU Lesser General Public License v2.1 only.
	LGPL21Only ID = "LGPL-2.1-only"

	// LGPL21OrLater is the GNU Lesser General Public License v2.1 or later.
	LGPL21OrLater ID = "LGPL-2.1-or-later"

	// LGPL30Only is the GNU Lesser General Public License v3.0 only.
	LGPL30Only ID = "LGPL-3.0-only"

	// LGPL30OrLater is the GNU Lesser General Public License v3.0 or later.
	LGPL30OrLater ID = "LGPL-3.0-or-later"

	// MIT is the MIT License.
	MIT ID = "MIT"

	// MIT0 is the MIT No Attribution license.
	MIT0 ID = "MIT-0"

	// MPL20 is the Mozilla Public License 2.0.
	MPL20 ID = "MPL-2.0"

	// MSPL is the Microsoft Public License.
	MSPL ID = "MS-PL"

	// MSRL is the Microsoft Reciprocal License.
	MSRL ID = "MS-RL"

	// MulanPSL20 is the Mulan Permissive Software License, Version 2.
	MulanPSL20 ID = "MulanPSL-2.0"

	// NCSA is the University of Illinois/NCSA Open Source License.
	NCSA ID = "NCSA"

	// OSL30 is the Open Software License 3.0.
	OSL30 ID = "OSL-3.0"

	// PostgreSQL is the PostgreSQL License.
	PostgreSQL ID = "PostgreSQL"

	// Unlicense is The Unlicense.
	Unlicense ID = "Unlicense"

	// UPL10 is the Universal Permissive License v1.0.
	UPL10 ID = "UPL-1.0"

	// Vim is the Vim License.
	Vim ID = "Vim"

	// WTFPL is the Do What The F*ck You Want To Public License.
	WTFPL ID = "WTFPL"

	// Zlib is the zlib License.
	Zlib ID = "Zlib"
)

// ids are the identifiers that ergon supports, in the order of the SPDX License List.
var ids = []ID{
	ZeroBSD, AFL30, AGPL30Only, AGPL30OrLater, Apache20, Artistic20, BlueOak100, BSD2Clause,
	BSD2ClausePatent, BSD3Clause, BSD3ClauseClear, BSD4Clause, BSL10, BUSL11, CC010, CECILL21,
	ECL20, EPL10, EPL20, EUPL11, EUPL12, GPL20Only, GPL20OrLater, GPL30Only, GPL30OrLater, ISC,
	LGPL21Only, LGPL21OrLater, LGPL30Only, LGPL30OrLater, MIT, MIT0, MPL20, MSPL, MSRL, MulanPSL20,
	NCSA, OSL30, PostgreSQL, Unlicense, UPL10, Vim, WTFPL, Zlib,
}

// IDs returns an iterator over the 44 identifiers that ergon supports, in the order of the SPDX
// License List, which sorts them without regard to case.
func IDs() iter.Seq[ID] {
	return slices.Values(ids)
}

// Valid reports whether id is one of the identifiers that ergon supports, in its canonical case.
// It reports false for an identifier of another case, such as "mit", and for an identifier that
// the SPDX License List deprecates, such as "GPL-3.0".
func (id ID) Valid() bool {
	return slices.Contains(ids, id)
}
