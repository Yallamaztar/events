package events

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func TailFileContext(ctx context.Context, path string, startAtEnd bool, eventsCh chan<- Event) error {
	const pollInterval = 150 * time.Millisecond
	const reopenRetry = 200 * time.Millisecond

	openFile := func() (*os.File, error) {
		return os.Open(path)
	}

	f, err := openFile()
	if err != nil {
		return err
	}
	defer f.Close()

	if startAtEnd {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return err
		}
	} else {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	buf := bufio.NewReader(f)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				stat, statErr := os.Stat(path)
				if statErr == nil {
					curStat, _ := f.Stat()
					if !os.SameFile(stat, curStat) || stat.Size() < currentOffset(f) {
						f.Close()
						var nf *os.File
						for {
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
							}
							nf, err = openFile()
							if err == nil {
								break
							}
							time.Sleep(reopenRetry)
						}
						f = nf
						buf = bufio.NewReader(f)
						continue
					}
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(pollInterval):
				}
				continue
			}
			return err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}

		ev, err := ParseEventLine(line)
		if err != nil {
			log.Printf("events: failed to parse event line: %v", err)
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case eventsCh <- ev:
		}
	}
}

func TailFile(path string, startAtEnd bool, eventsCh chan<- Event) error {
	return TailFileContext(context.Background(), path, startAtEnd, eventsCh)
}

func currentOffset(f *os.File) int64 {
	off, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0
	}
	return off
}
