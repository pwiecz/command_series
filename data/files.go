package data

import "bufio"
import "io"

func UnpackFile(data io.Reader) ([]byte, error) {
	var header [5]byte
	if _, err := io.ReadFull(data, header[:]); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(data)
	var decodedData []byte
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
	return decodedData, nil
}
