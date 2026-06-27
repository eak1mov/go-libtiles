package copier

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"sync/atomic"

	"github.com/eak1mov/go-libtiles/tile"
	"golang.org/x/sync/errgroup"
)

type options struct {
	BufferSize  int
	Concurrency int
}

type Option func(*options)

// BufferSize is the maximum RAM size used for a single batch.
func BufferSize(bufferSize int) Option {
	return func(c *options) { c.BufferSize = bufferSize }
}

// Concurrency is the number of goroutines for parallel reading.
func Concurrency(concurrency int) Option {
	return func(c *options) { c.Concurrency = concurrency }
}

const DefaultBufferSize = 8 << 30 // 8 GiB
const DefaultConcurrency = 32

// Copier encapsulates the logic for batching and concurrent reading.
type Copier struct {
	readBuffer  []byte
	concurrency int
}

func New(opts ...Option) *Copier {
	copts := options{
		BufferSize:  DefaultBufferSize,
		Concurrency: DefaultConcurrency,
	}
	for _, opt := range opts {
		opt(&copts)
	}
	return &Copier{
		readBuffer:  make([]byte, copts.BufferSize),
		concurrency: copts.Concurrency,
	}
}

// Copy concurrently reads locations from src and sequentially writes them to dst.
func (c *Copier) Copy(ctx context.Context, dst io.Writer, src io.ReaderAt, locations []tile.Location) error {
	for _, batch := range makeBatches(locations, uint64(len(c.readBuffer))) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("copy interrupted: %w", ctx.Err())
		default:
		}

		slices.SortFunc(batch.ReadRequests, func(a, b readRequest) int {
			return cmp.Compare(a.Location.Offset, b.Location.Offset)
		})

		batchBuf := c.readBuffer[:batch.TotalLength]

		if err := read(ctx, batch.ReadRequests, src, batchBuf, c.concurrency); err != nil {
			return fmt.Errorf("read failed: %w", err)
		}

		_, err := dst.Write(batchBuf)
		if err != nil {
			return fmt.Errorf("write failed: %w", err)
		}
	}
	return nil
}

// readBatch represents a single prepared batch of requests that fits into the memory limit.
type readBatch struct {
	ReadRequests []readRequest
	TotalLength  uint64
}

// readRequest is linking a source location with its destination position in the RAM buffer.
type readRequest struct {
	Location     tile.Location
	BufferOffset uint64
}

func makeBatches(locations []tile.Location, bufferLen uint64) []readBatch {
	var batches []readBatch
	var currentReqs []readRequest
	var currentOffset uint64

	for _, l := range locations {
		if currentOffset+l.Length > bufferLen {
			batches = append(batches, readBatch{
				ReadRequests: currentReqs,
				TotalLength:  currentOffset,
			})
			currentReqs = nil
			currentOffset = 0
		}

		currentReqs = append(currentReqs, readRequest{
			Location:     l,
			BufferOffset: currentOffset,
		})
		currentOffset += l.Length
	}

	if len(currentReqs) > 0 {
		batches = append(batches, readBatch{
			ReadRequests: currentReqs,
			TotalLength:  currentOffset,
		})
	}

	return batches
}

func read(ctx context.Context, requests []readRequest, src io.ReaderAt, dstBuf []byte, concurrency int) error {
	group, gCtx := errgroup.WithContext(ctx)

	var reqCounter atomic.Int64

	for range concurrency {
		group.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				default:
				}

				reqIdx := int(reqCounter.Add(1)) - 1
				if reqIdx >= len(requests) {
					return nil
				}

				req := requests[reqIdx]
				reqBuf := dstBuf[req.BufferOffset:][:req.Location.Length]

				_, err := src.ReadAt(reqBuf, int64(req.Location.Offset))
				if err != nil {
					return err
				}
			}
		})
	}

	return group.Wait()
}
