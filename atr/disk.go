package atr

import "bytes"
import "fmt"
import "io"
import "os"

type SectorReader interface {
	/* 1-based sector number */
	ReadSector(num int) ([]byte, error)
}

const (
	ATR_MAGIC1 = byte(0x96)
	ATR_MAGIC2 = byte(0x02)
)

type atrSectorReader struct {
	filename    string
	sectorSize  int
	sectorCount int
}

func NewAtrSectorReader(filename string) (SectorReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Cannot open file %s, %v", filename, err)
	}
	defer file.Close()
	var atrHeader [16]byte // 8 header bytes + 8 reserved bytes
	if _, err := io.ReadFull(file, atrHeader[:]); err != nil {
		return nil, fmt.Errorf("Cannot read atr file header from file %s, %v", filename, err)
	}
	if atrHeader[0] != ATR_MAGIC1 || atrHeader[1] != ATR_MAGIC2 {
		return nil, fmt.Errorf("%s is not an atr file", filename)
	}
	sectorReader := &atrSectorReader{}
	sectorReader.filename = filename
	sectorReader.sectorSize = int(atrHeader[4]) + (int(atrHeader[5]) << 8)
	imageSize := (int(atrHeader[2]) + (int(atrHeader[3]) << 8) +
		(int(atrHeader[6]) << 16) + (int(atrHeader[7]) << 24)) * 16
	sectorReader.sectorCount = 3 + (imageSize - 3*128) / sectorReader.sectorSize
	if sectorReader.sectorSize == 256 {
		sectorReader.sectorCount = (sectorReader.sectorCount + 3) / 2
	}
	return sectorReader, nil
}

func (r *atrSectorReader) ReadSector(sector int) ([]byte, error) {
	if sector < 1 || sector > r.sectorCount {
		return nil, fmt.Errorf("Invalid sector number %d", sector)
	}
	file, err := os.Open(r.filename)
	if err != nil {
		return nil, fmt.Errorf("Cannot open file %s, %v", r.filename, err)
	}
	defer file.Close()
	offset := 16 /* size of the header */
	if sector <= 4 {
		offset += (sector - 1) * 128
	} else {
		offset += 3*128 + (sector-4)*r.sectorSize
	}
	if _, err := file.Seek(int64(offset), 0); err != nil {
		return nil, fmt.Errorf("Cannot seek to position %d int file %s, %v", offset, r.filename, err)
	}
	sectorSize := r.sectorSize
	if sector <= 3 {
		sectorSize = 128
	}
	data := make([]byte, sectorSize)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, err
	}
	return data, nil
}

type AtrFile struct {
	Name   string
	Index  int
	Attrib byte
	Size   int
	Start  int
}

const (
	DELETED = 0x80
)

func GetDirectory(reader SectorReader) ([]AtrFile, error) {
	var res []AtrFile
	for sectorNum := 360; sectorNum < 368; sectorNum++ {
		sectorData, err := reader.ReadSector(sectorNum)
		if err != nil {
			return nil, err
		}

		for entryStart := 0; entryStart+16 <= len(sectorData); entryStart += 16 {
			entryData := sectorData[entryStart : entryStart+16]
			if entryData[0] == 0 || entryData[0]&DELETED != 0 || entryData[0]&0x40 == 0 {
				continue
			}
			var atrFile AtrFile
			name := bytes.TrimRight(entryData[5:13], " ")
			extension := bytes.TrimRight(entryData[13:16], " ")
			atrFile.Name = string(name) + "." + string(extension)
			atrFile.Index = len(res)
			atrFile.Attrib = entryData[0]
			atrFile.Size = int(entryData[2])*256 + int(entryData[1])
			atrFile.Start = int(entryData[4])*256 + int(entryData[3])
			res = append(res, atrFile)
		}
	}
	return res, nil
}

func ReadFile(reader SectorReader, filename string) ([]byte, error) {
	dir, err := GetDirectory(reader)
	if err != nil {
		return nil, err
	}
	for _, entry := range dir {
		if entry.Name != filename {
			// todo: check attrs if it was not deleted?
			continue
		}
		var content []byte
		sectorNum := entry.Start
		for {
			sector, err := reader.ReadSector(sectorNum)
			if err != nil {
				return nil, err
			}
			if len(sector) < 3 {
				return nil, fmt.Errorf("Unsupported sector size: %d", len(sector))
			}
			fileIndex := int(sector[len(sector)-3] >> 2)
			if fileIndex != entry.Index {
				return nil, fmt.Errorf("File# mismatch, %d != %d", fileIndex, entry.Index)
			}

			dataLen := int(sector[len(sector)-1] & 0x7f)
			if dataLen > len(sector)-3 {
				return nil, fmt.Errorf("Invalid data length of sector: %d", dataLen)
			}
			for i := 0; i < dataLen; i++ {
				content = append(content, sector[i])
			}
			if sector[len(sector)-1]&0x80 != 0 {
				return content, nil
			}
			sectorNum = int(sector[len(sector)-3]&0b11)<<8 + int(sector[len(sector)-2])
			if sectorNum == 0 {
				return content, nil
			}
		}
	}
	return nil, fmt.Errorf("File %s not found", filename)
}
