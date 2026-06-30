package mediamtx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

type ManifestInput struct {
	StreamID          string
	StartedAt         time.Time
	SegmentDuration   time.Duration
	SegmentPaths      []string
	IssuedNonces      []core.LivenessNonce
	DeliveryReceipts  []core.DeliveryReceipt
	SampledFrames     []core.SampledFrame
	WorkerPushedBytes map[string]int64
}

func BuildStreamManifest(input ManifestInput) (core.StreamManifest, error) {
	if strings.TrimSpace(input.StreamID) == "" {
		return core.StreamManifest{}, errors.New("stream id is required")
	}
	if input.StartedAt.IsZero() {
		return core.StreamManifest{}, errors.New("started_at is required")
	}
	if input.SegmentDuration <= 0 {
		return core.StreamManifest{}, errors.New("segment duration is required")
	}
	if len(input.SegmentPaths) == 0 {
		return core.StreamManifest{}, errors.New("segment paths are required")
	}

	segments := make([]core.Segment, 0, len(input.SegmentPaths))
	segmentBytes := make([][]byte, 0, len(input.SegmentPaths))
	for idx, path := range input.SegmentPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return core.StreamManifest{}, fmt.Errorf("read segment %d: %w", idx, err)
		}
		sum := sha256.Sum256(data)
		segments = append(segments, core.Segment{
			Index:      idx,
			DurationMS: input.SegmentDuration.Milliseconds(),
			Bytes:      int64(len(data)),
			SHA256:     "sha256:" + hex.EncodeToString(sum[:]),
			Provenance: core.ProvenanceHostVerified,
		})
		segmentBytes = append(segmentBytes, data)
	}

	nonces := make([]core.LivenessNonce, 0, len(input.IssuedNonces))
	for _, issued := range input.IssuedNonces {
		reflected := issued
		reflected.ReflectedInSeq = -1
		for idx, data := range segmentBytes {
			if strings.Contains(string(data), issued.Nonce) {
				reflected.ReflectedInSeq = idx
				break
			}
		}
		if reflected.ReflectedInSeq >= 0 {
			nonces = append(nonces, reflected)
		}
	}

	receipts := make([]core.DeliveryReceipt, 0, len(input.DeliveryReceipts))
	for _, receipt := range input.DeliveryReceipts {
		if receipt.SeqStart == receipt.SeqEnd && receipt.SeqStart >= 0 && receipt.SeqStart < len(segments) {
			receipt.SHA256 = segments[receipt.SeqStart].SHA256
			receipt.Bytes = segments[receipt.SeqStart].Bytes
		}
		receipts = append(receipts, receipt)
	}

	return core.StreamManifest{
		StreamID:                         input.StreamID,
		StartedAt:                        input.StartedAt,
		LivenessNonces:                   nonces,
		Segments:                         segments,
		SampledFrames:                    append([]core.SampledFrame(nil), input.SampledFrames...),
		DeliveryReceipts:                 receipts,
		AttestedPushedBytesByDestination: cloneByteMap(input.WorkerPushedBytes),
		WorkerReportedDeliveredBytes:     sumReceiptBytes(input.DeliveryReceipts),
		Discontinuities:                  nil,
		DroppedFrames:                    0,
	}, nil
}

func cloneByteMap(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sumReceiptBytes(receipts []core.DeliveryReceipt) int64 {
	var total int64
	for _, receipt := range receipts {
		total += receipt.Bytes
	}
	return total
}
