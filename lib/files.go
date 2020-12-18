package lib

import "bufio"
import "io"

func UnpackFile(data io.Reader) ([]byte, error) {
	var header [5]byte
	if _, err := io.ReadFull(data, header[:]); err != nil {
		return nil, err
	}
	// TODO: understand what's this number. It's some kind of an upper bound
	// of the decoded size.
	expectedSize := 256*int(header[4]) + int(header[3]) - 256*int(header[2]) + int(header[1]) + 1
	reader := bufio.NewReader(data)
	decodedData := make([]byte, 0, expectedSize)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		if b != header[0] {
			decodedData = append(decodedData, b)
		} else {
			var valueCount [2]byte
			if _, err := io.ReadFull(reader, valueCount[:]); err != nil {
				return nil, err
			}
			count := int(valueCount[1]) + 4
			if valueCount[1] == 0xff {
				num, err := reader.ReadByte()
				if err != nil {
					return nil, err
				}
				count += int(num)
			}
			for i := 0; i < count; i++ {
				decodedData = append(decodedData, valueCount[0])
			}
		}
	}
	for len(decodedData) < expectedSize {
		decodedData = append(decodedData, 0)
	}
	return decodedData, nil
}
