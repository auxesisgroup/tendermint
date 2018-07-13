package ed25519

import (
	"bytes"
	"crypto/subtle"
	"fmt"

	"github.com/tendermint/ed25519"
	"github.com/tendermint/ed25519/extra25519"
	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tmlibs/common"
)

//-------------------------------------

var _ crypto.PrivKey = PrivKeyEd25519{}

const (
	Ed25519PrivKeyAminoRoute   = "tendermint/PrivKeyEd25519"
	Ed25519PubKeyAminoRoute    = "tendermint/PubKeyEd25519"
	Ed25519SignatureAminoRoute = "tendermint/SignatureEd25519"
)

var cdc = amino.NewCodec()

func init() {
	// NOTE: It's important that there be no conflicts here,
	// as that would change the canonical representations,
	// and therefore change the address.
	// TODO: Add feature to go-amino to ensure that there
	// are no conflicts.
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(PubKeyEd25519{},
		Ed25519PubKeyAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(PrivKeyEd25519{},
		Ed25519PrivKeyAminoRoute, nil)

	cdc.RegisterInterface((*crypto.Signature)(nil), nil)
	cdc.RegisterConcrete(SignatureEd25519{},
		Ed25519SignatureAminoRoute, nil)
}

// Implements crypto.PrivKey
type PrivKeyEd25519 [64]byte

func (privKey PrivKeyEd25519) Bytes() []byte {
	return cdc.MustMarshalBinaryBare(privKey)
}

func (privKey PrivKeyEd25519) Sign(msg []byte) (crypto.Signature, error) {
	privKeyBytes := [64]byte(privKey)
	signatureBytes := ed25519.Sign(&privKeyBytes, msg)
	return SignatureEd25519(*signatureBytes), nil
}

func (privKey PrivKeyEd25519) PubKey() crypto.PubKey {
	privKeyBytes := [64]byte(privKey)
	pubBytes := *ed25519.MakePublicKey(&privKeyBytes)
	return PubKeyEd25519(pubBytes)
}

// Equals - you probably don't need to use this.
// Runs in constant time based on length of the keys.
func (privKey PrivKeyEd25519) Equals(other crypto.PrivKey) bool {
	if otherEd, ok := other.(PrivKeyEd25519); ok {
		return subtle.ConstantTimeCompare(privKey[:], otherEd[:]) == 1
	} else {
		return false
	}
}

func (privKey PrivKeyEd25519) ToCurve25519() *[32]byte {
	keyCurve25519 := new([32]byte)
	privKeyBytes := [64]byte(privKey)
	extra25519.PrivateKeyToCurve25519(keyCurve25519, &privKeyBytes)
	return keyCurve25519
}

// Deterministically generates new priv-key bytes from key.
func (privKey PrivKeyEd25519) Generate(index int) PrivKeyEd25519 {
	bz, err := cdc.MarshalBinaryBare(struct {
		PrivKey [64]byte
		Index   int
	}{privKey, index})
	if err != nil {
		panic(err)
	}
	newBytes := crypto.Sha256(bz)
	newKey := new([64]byte)
	copy(newKey[:32], newBytes)
	ed25519.MakePublicKey(newKey)
	return PrivKeyEd25519(*newKey)
}

func GenPrivKeyEd25519() PrivKeyEd25519 {
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], crypto.CRandBytes(32))
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes)
}

// NOTE: secret should be the output of a KDF like bcrypt,
// if it's derived from user input.
func GenPrivKeyEd25519FromSecret(secret []byte) PrivKeyEd25519 {
	privKey32 := crypto.Sha256(secret) // Not Ripemd160 because we want 32 bytes.
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], privKey32)
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes)
}

//-------------------------------------

var _ crypto.PubKey = PubKeyEd25519{}

const PubKeyEd25519Size = 32

// Implements PubKeyInner
type PubKeyEd25519 [PubKeyEd25519Size]byte

// Address is the SHA256-20 of the raw pubkey bytes.
func (pubKey PubKeyEd25519) Address() crypto.Address {
	return crypto.Address(tmhash.Sum(pubKey[:]))
}

func (pubKey PubKeyEd25519) Bytes() []byte {
	bz, err := cdc.MarshalBinaryBare(pubKey)
	if err != nil {
		panic(err)
	}
	return bz
}

func (pubKey PubKeyEd25519) VerifyBytes(msg []byte, sig_ crypto.Signature) bool {
	// make sure we use the same algorithm to sign
	sig, ok := sig_.(SignatureEd25519)
	if !ok {
		return false
	}
	pubKeyBytes := [PubKeyEd25519Size]byte(pubKey)
	sigBytes := [SignatureEd25519Size]byte(sig)
	return ed25519.Verify(&pubKeyBytes, msg, &sigBytes)
}

// For use with golang/crypto/nacl/box
// If error, returns nil.
func (pubKey PubKeyEd25519) ToCurve25519() *[PubKeyEd25519Size]byte {
	keyCurve25519, pubKeyBytes := new([PubKeyEd25519Size]byte), [PubKeyEd25519Size]byte(pubKey)
	ok := extra25519.PublicKeyToCurve25519(keyCurve25519, &pubKeyBytes)
	if !ok {
		return nil
	}
	return keyCurve25519
}

func (pubKey PubKeyEd25519) String() string {
	return fmt.Sprintf("PubKeyEd25519{%X}", pubKey[:])
}

func (pubKey PubKeyEd25519) Equals(other crypto.PubKey) bool {
	if otherEd, ok := other.(PubKeyEd25519); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	} else {
		return false
	}
}

//-------------------------------------

var _ crypto.Signature = SignatureEd25519{}

const SignatureEd25519Size = 64

// Implements crypto.Signature
type SignatureEd25519 [SignatureEd25519Size]byte

func (sig SignatureEd25519) Bytes() []byte {
	bz, err := cdc.MarshalBinaryBare(sig)
	if err != nil {
		panic(err)
	}
	return bz
}

func (sig SignatureEd25519) IsZero() bool { return len(sig) == 0 }

func (sig SignatureEd25519) String() string { return fmt.Sprintf("/%X.../", common.Fingerprint(sig[:])) }

func (sig SignatureEd25519) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureEd25519); ok {
		return subtle.ConstantTimeCompare(sig[:], otherEd[:]) == 1
	} else {
		return false
	}
}

func SignatureEd25519FromBytes(data []byte) crypto.Signature {
	var sig SignatureEd25519
	copy(sig[:], data)
	return sig
}
