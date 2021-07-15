package zkprm

import (
	"crypto/rand"
	"math/big"

	"github.com/taurusgroup/cmp-ecdsa/pkg/hash"
	"github.com/taurusgroup/cmp-ecdsa/pkg/math/arith"
	"github.com/taurusgroup/cmp-ecdsa/pkg/math/sample"
	"github.com/taurusgroup/cmp-ecdsa/pkg/params"
	"github.com/taurusgroup/cmp-ecdsa/pkg/pedersen"
)

type (
	Public struct {
		Pedersen *pedersen.Parameters
	}
	Private struct {
		Lambda, Phi *big.Int
	}
)

func (p Proof) IsValid(public Public) bool {
	if len(*p.A) != params.StatParam || len(*p.Z) != params.StatParam {
		return false
	}
	if !arith.IsValidModN(public.Pedersen.N, *p.A...) {
		return false
	}
	if !arith.IsValidModN(public.Pedersen.N, *p.Z...) {
		return false
	}
	return true
}

// NewProof generates a proof that:
// s = t^lambda (mod N)
func NewProof(hash *hash.Hash, public Public, private Private) *Proof {
	n := public.Pedersen.N
	phi := private.Phi

	a := make([]*big.Int, params.StatParam)
	A := make([]*big.Int, params.StatParam)

	for i := 0; i < params.StatParam; i++ {
		// aᵢ ∈ mod ϕ(N)
		a[i] = sample.IntervalLN(rand.Reader)
		a[i].Mod(a[i], phi)

		// Aᵢ = tᵃ mod N
		A[i] = new(big.Int).Exp(public.Pedersen.T, a[i], n)
	}

	es := challenge(hash, public, A)

	Z := make([]*big.Int, params.StatParam)
	for i := 0; i < params.StatParam; i++ {
		z := a[i]
		if es[i] {
			z.Add(z, private.Lambda)
			z.Mod(z, phi)
		}
		Z[i] = z
	}

	return &Proof{
		A: &A,
		Z: &Z,
	}
}

func (p *Proof) Verify(hash *hash.Hash, public Public) bool {
	if !p.IsValid(public) {
		return false
	}

	if err := public.Pedersen.Validate(); err != nil {
		return false
	}

	n, s, t := public.Pedersen.N, public.Pedersen.S, public.Pedersen.T

	es := challenge(hash, public, *p.A)

	var lhs, rhs big.Int
	one := big.NewInt(1)
	for i := 0; i < params.StatParam; i++ {
		z := (*p.Z)[i]
		a := (*p.A)[i]

		if a.Cmp(one) == 0 {
			return false
		}

		lhs.Exp(t, z, n)
		if es[i] {
			rhs.Mul(a, s)
			rhs.Mod(&rhs, n)
		} else {
			rhs.Set(a)
		}

		if lhs.Cmp(&rhs) != 0 {
			return false
		}
	}
	return true
}

func challenge(hash *hash.Hash, public Public, A []*big.Int) []bool {
	_, _ = hash.WriteAny(public.Pedersen.N, public.Pedersen.S, public.Pedersen.T)
	for _, a := range A {
		_, _ = hash.WriteAny(a)
	}

	tmpBytes := make([]byte, params.StatParam)
	hash.ReadBytes(tmpBytes)

	out := make([]bool, params.StatParam)
	for i := range out {
		b := (tmpBytes[i] & 1) == 1
		out[i] = b
	}

	return out
}