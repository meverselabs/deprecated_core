package level

import (
	"bytes"
	"sync"

	"git.fleta.io/fleta/common/hash"
)

const hashPerLevel = 16
const levelHashAppender = "fletablockchain"

// Hash TODO
func Hash(hashes []hash.Hash256) (hash.Hash256, error) {
	if len(hashes) > hashPerLevel {
		return hash.Hash256{}, ErrExceedHashCount
	}

	var buffer bytes.Buffer
	var EmptyHash hash.Hash256
	for i := 0; i < hashPerLevel; i++ {
		if i < len(hashes) {
			if _, err := hashes[i].WriteTo(&buffer); err != nil {
				return hash.Hash256{}, err
			}
		} else if _, err := EmptyHash.WriteTo(&buffer); err != nil {
			return hash.Hash256{}, err
		}
		if i < len(levelHashAppender) {
			if err := buffer.WriteByte(byte(levelHashAppender[i])); err != nil {
				return hash.Hash256{}, err
			}
		}
	}
	return hash.Hash(buffer.Bytes()), nil
}

// BuildLevel TODO
func BuildLevel(hashes []hash.Hash256) ([]hash.Hash256, error) {
	LvCnt := len(hashes) / hashPerLevel
	if len(hashes)%hashPerLevel != 0 {
		LvCnt++
	}

	var wg sync.WaitGroup
	errs := make(chan error, LvCnt)
	LvHashes := make([]hash.Hash256, LvCnt)
	for i := 0; i < LvCnt; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			last := (idx + 1) * hashPerLevel
			if last > len(hashes) {
				last = len(hashes)
			}
			h, err := Hash(hashes[idx*hashPerLevel : last])
			if err != nil {
				errs <- err
				return
			}
			LvHashes[idx] = h
		}(i)
	}
	wg.Wait()
	if len(errs) > 0 {
		err := <-errs
		return nil, err
	}
	return LvHashes, nil
}

// BuildLevelRoot TODO
func BuildLevelRoot(txHashes []hash.Hash256) (hash.Hash256, error) {
	if len(txHashes) > 65535 {
		return hash.Hash256{}, ErrExceedHashCount
	}
	if len(txHashes) == 0 {
		return hash.Hash256{}, nil
	}

	lv3, err := BuildLevel(txHashes)
	if err != nil {
		return hash.Hash256{}, err
	}
	lv2, err := BuildLevel(lv3)
	if err != nil {
		return hash.Hash256{}, err
	}
	lv1, err := BuildLevel(lv2)
	if err != nil {
		return hash.Hash256{}, err
	}
	h, err := Hash(lv1)
	if err != nil {
		return hash.Hash256{}, err
	}
	return h, nil
}
