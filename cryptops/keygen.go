////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cryptops wraps various cryptographic operations around a generic interface.
// Operations include but are not limited to: key generation, ElGamal, multiplication, etc.
package cryptops

import (
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// Wraps existing keygen operations in the cmix package
type KeygenPrototype func(group *cyclic.Group,
	salt []byte, roundID id.Round, symmetricKey, generatedKey *cyclic.Int)

// KeyGen implements the cmix.NodeKeyGen(() within the cryptops interface
var Keygen KeygenPrototype = cmix.NodeKeyGen

// GetName returns the function name for debugging.
func (KeygenPrototype) GetName() string {
	return "Keygen"
}

// GetInputSize returns the input size; used in safety checks.
func (KeygenPrototype) GetInputSize() uint32 {
	return 1
}
