// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
)

// precompileTimingTracer measures wall-clock time spent in specific precompiles during a block.
// It listens to EVM call enter/exit events and accumulates durations per targeted precompile.
type precompileTimingTracer struct {
	// startByDepth holds start timestamps for active precompile calls keyed by call depth.
	startByDepth map[int]time.Time
	// kindByDepth holds the bucket name for the active precompile call keyed by call depth.
	kindByDepth map[int]string
	// totalByKind accumulates total time per precompile kind for the current block.
	totalByKind map[string]time.Duration
}

func newPrecompileTimingTracer() *precompileTimingTracer {
	return &precompileTimingTracer{
		startByDepth: make(map[int]time.Time),
		kindByDepth:  make(map[int]string),
		totalByKind:  make(map[string]time.Duration),
	}
}

// address constants for targeted precompiles
var (
	addrModExp = common.BytesToAddress([]byte{0x05}) // EIP-198/2565
	addrBnAdd  = common.BytesToAddress([]byte{0x06}) // BN256 ECADD
	addrBnMul  = common.BytesToAddress([]byte{0x07}) // BN256 ECMUL
	addrBnPair = common.BytesToAddress([]byte{0x08}) // BN256 Pairing
	addrKZG    = common.BytesToAddress([]byte{0x0a}) // EIP-4844 KZG point evaluation
)

func (t *precompileTimingTracer) classify(addr common.Address) (string, bool) {
	switch addr {
	case addrModExp:
		return "modexp", true
	case addrBnAdd:
		return "bn256_add", true
	case addrBnMul:
		return "bn256_mul", true
	case addrBnPair:
		return "bn256_pairing", true
	case addrKZG:
		return "kzg_point_eval", true
	default:
		return "", false
	}
}

func (t *precompileTimingTracer) onEnter(depth int, to common.Address) {
	if kind, ok := t.classify(to); ok {
		// Record start time for this depth
		t.startByDepth[depth] = time.Now()
		t.kindByDepth[depth] = kind
	}
}

func (t *precompileTimingTracer) onExit(depth int) {
	if start, ok := t.startByDepth[depth]; ok {
		kind := t.kindByDepth[depth]
		dur := time.Since(start)
		t.totalByKind[kind] += dur
		delete(t.startByDepth, depth)
		delete(t.kindByDepth, depth)
	}
}

// withPrecompileTiming returns a Hooks instance that augments the provided base hooks
// with precompile timing. All existing hooks are preserved; OnEnter/OnExit are wrapped
// to invoke the timer and then the base hook (if present).
func withPrecompileTiming(base *tracing.Hooks, timer *precompileTimingTracer) *tracing.Hooks {
	var combined tracing.Hooks
	if base != nil {
		combined = *base // copy existing hooks
	}
	prevEnter := combined.OnEnter
	combined.OnEnter = func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
		timer.onEnter(depth, to)
		if prevEnter != nil {
			prevEnter(depth, typ, from, to, input, gas, value)
		}
	}
	prevExit := combined.OnExit
	combined.OnExit = func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
		timer.onExit(depth)
		if prevExit != nil {
			prevExit(depth, output, gasUsed, err, reverted)
		}
	}
	return &combined
}

// snapshotTotals returns current totals as millisecond values for logging.
func (t *precompileTimingTracer) snapshotTotals() map[string]int64 {
	return map[string]int64{
		"modexp_ms":         t.totalByKind["modexp"].Milliseconds(),
		"bn256_add_ms":      t.totalByKind["bn256_add"].Milliseconds(),
		"bn256_mul_ms":      t.totalByKind["bn256_mul"].Milliseconds(),
		"bn256_pairing_ms":  t.totalByKind["bn256_pairing"].Milliseconds(),
		"kzg_point_eval_ms": t.totalByKind["kzg_point_eval"].Milliseconds(),
	}
}
