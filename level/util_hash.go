package level

import (
	"bytes"
	"sync"

	"github.com/fletaio/common/hash"
)

const hashPerLevel = 16
const levelHashAppender = "fletablockchain"

// Hash16 returns Hash(x1,'f',x2,'l',...,x16)
func Hash16(hashes []hash.Hash256) (hash.Hash256, error) {
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
	return hash.DoubleHash(buffer.Bytes()), nil
}

func buildLevel(hashes []hash.Hash256) ([]hash.Hash256, error) {
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
			h, err := Hash16(hashes[idx*hashPerLevel : last])
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

// BuildLevelRoot returns the level root hash
func BuildLevelRoot(hashes []hash.Hash256) (hash.Hash256, error) {
	if len(hashes) > 65536 {
		return hash.Hash256{}, ErrExceedHashCount
	}
	if len(hashes) == 0 {
		return hash.Hash256{}, ErrInvalidHashCount
	}

	lv3, err := buildLevel(hashes)
	if err != nil {
		return hash.Hash256{}, err
	}
	lv2, err := buildLevel(lv3)
	if err != nil {
		return hash.Hash256{}, err
	}
	lv1, err := buildLevel(lv2)
	if err != nil {
		return hash.Hash256{}, err
	}
	h, err := Hash16(lv1)
	if err != nil {
		return hash.Hash256{}, err
	}
	return h, nil
}
