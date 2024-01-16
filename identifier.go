package main

import (
	`fmt`
	`path/filepath`
)

var sliceRanges = []int{1, 3, 6, 10, 15, 21, 28, 36, 45, 64}

func checksum_to_path(checksum string) (string, error) {
	if len(checksum) != 64 {
		return checksum, fmt.Errorf("invalid checksum length. must be %d bytes", len(checksum))
	}
	var paths []string

	for i := 0; i < len(sliceRanges); i++ {
		r := sliceRanges[i]
		if r > len(checksum) {
			break // Avoid slicing beyond the length of the checksum
		}
		paths = append(paths, checksum[:r])
	}

	return filepath.Join(paths...), nil
}
