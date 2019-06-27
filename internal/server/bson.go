package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
)

const (
	byteValue   byte = 0x01
	shortValue  byte = 0x02
	intValue    byte = 0x03
	floatValue  byte = 0x04
	doubleValue byte = 0x05
	nullValue   byte = 0x06
	stringValue byte = 0x07
	stridxValue byte = 0x08
	binaryValue byte = 0x09
	arrayValue  byte = 0x0a
	objectValue byte = 0x0b
	endValue    byte = 0x0e
)

const (
	extern  int16 = 0
	encry   byte  = 0
	version byte  = 10
	cmdType byte  = 0
)

//Bson bsondata
type Bson struct {
	bsonGetIdx map[byte]string
	bsonStrIdx map[string]byte
}

//NewBson new
func NewBson() *Bson {
	b := Bson{
		bsonGetIdx: make(map[byte]string),
		bsonStrIdx: make(map[string]byte),
	}
	if err := b.initStrTab(); err != nil {
		return nil
	}
	return &b
}

func (b *Bson) initStrTab() error {
	const dataFile = "../../configs/strTab.bson"
	_, filename, _, _ := runtime.Caller(1)
	datapath := path.Join(path.Dir(filename), dataFile)

	f, err := os.Open(datapath)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	i := byte(0)
	for ; ; i++ {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		b.bsonGetIdx[i] = line
		if err != nil {
			if err == io.EOF {
				for k, v := range b.bsonGetIdx {
					b.bsonStrIdx[v] = k
				}
				return nil
			}
			return err
		}
	}
}

//Set map[string] inerface{}
func (b *Bson) Set(data map[string]interface{}, cmdid int16) []byte {
	bsonbuf := new(bytes.Buffer)
	obj := b.transFromMap(&data)
	len := int32(len(obj.Bytes()) + 11)
	value := new(bytes.Buffer)
	binary.Write(value, binary.LittleEndian, &len)
	bsonbuf.Write(value.Bytes())
	value.Reset()
	binary.Write(value, binary.LittleEndian, &cmdid)
	bsonbuf.Write(value.Bytes())
	value.Reset()
	bsonbuf.WriteByte(1)
	bsonbuf.WriteByte(1)
	bsonbuf.WriteByte(0)
	extLen := int16(0)
	binary.Write(value, binary.LittleEndian, &extLen)
	bsonbuf.Write(value.Bytes())
	value.Reset()
	bsonbuf.Write(obj.Bytes())
	// fmt.Println(bsonbuf.Bytes())
	return bsonbuf.Bytes()
}

func (b *Bson) transFromMap(data *map[string]interface{}) (obj bytes.Buffer) {
	obj.WriteByte(objectValue)
	for key, value := range *data {
		obj.WriteByte(stridxValue)
		obj.WriteByte(b.bsonStrIdx[key])
		switch v := value.(type) {
		case int16:
			obj.WriteByte(shortValue)
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, &v)
			obj.Write(buf.Bytes())
		case int32:
			obj.WriteByte(intValue)
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, &v)
			obj.Write(buf.Bytes())
		case int:
			obj.WriteByte(intValue)
			buf := new(bytes.Buffer)
			v32 := int32(v)
			binary.Write(buf, binary.LittleEndian, &v32)
			obj.Write(buf.Bytes())
		case string:
			obj.WriteByte(stringValue)
			obj.WriteString(v)
			obj.WriteByte(0)
		case byte:
			obj.WriteByte(byteValue)
			obj.WriteByte(v)
		case map[string]interface{}:
			by := b.transFromMap(&v)
			obj.Write(by.Bytes())
		case []interface{}:
			obj.WriteByte(arrayValue)
			if len(v) != 0 {
				for _, arr := range v {
					switch t := arr.(type) {
					case string:
						obj.WriteByte(stringValue)
						obj.WriteString(t)
						obj.WriteByte(0)
					case int:
						obj.WriteByte(intValue)
						buf := new(bytes.Buffer)
						v32 := int32(t)
						binary.Write(buf, binary.LittleEndian, &v32)
						obj.Write(buf.Bytes())
					case byte:
						obj.WriteByte(byteValue)
						obj.WriteByte(t)
					default:
						log.Println("set single array type need to repaire")
						obj.WriteByte(nullValue)
					}
				}
				obj.WriteByte(endValue)
			} else {
				obj.WriteByte(objectValue)
				obj.WriteByte(endValue)
			}
			obj.WriteByte(endValue)
		case []map[string]interface{}:
			obj.WriteByte(arrayValue)
			if len(v) != 0 {
				for _, arr := range v {
					by := b.transFromMap(&arr)
					obj.Write(by.Bytes())
				}
			} else {
				obj.WriteByte(objectValue)
				obj.WriteByte(endValue)
			}
			obj.WriteByte(endValue)
		default:
			obj.WriteByte(nullValue)
		}
	}
	obj.WriteByte(endValue)
	return
}

// Get data
func (b *Bson) Get(buf []byte) (map[string]interface{}, int, error) {
	index := 0
	var data = make(map[string]interface{})
	if buf[index] != objectValue {
		return nil, 0, errors.New("data error")
	}
	index++
	for {
		if buf[index] == stridxValue {
			index++
			name := b.bsonGetIdx[buf[index]]
			index++
			switch buf[index] {
			case stringValue:
				index++
				strCount := 0
				startIndex := index
				for buf[index] != 0 {
					strCount++
					index++
				}
				by := bytes.NewBuffer(buf[startIndex : startIndex+strCount])
				data[name] = by.String()
				index++
			case intValue:
				index++
				var v int32
				by := bytes.NewBuffer(buf[index : index+4])
				binary.Read(by, binary.LittleEndian, &v)
				data[name] = int(v)
				index += 4
			case arrayValue:
				index++
				if buf[index] == endValue {
					data[name] = []interface{}{}
					index++
				} else {

					if buf[index] == objectValue {
						arryData, endindex, err := b.Get(buf[index:])
						if err != nil {
							return nil, 0, errors.New("arry error")
						}
						data[name] = arryData
						index += endindex + 1
					} else {
						arryData := []interface{}{}
						for buf[index] != endValue {
							switch buf[index] {
							case stringValue:
								index++
								strCount := 0
								startIndex := index
								for buf[index] != 0 {
									strCount++
									index++
								}
								by := bytes.NewBuffer(buf[startIndex : startIndex+strCount])
								arryData = append(arryData, by.String())
								index++
							case intValue:
								index++
								var v int32
								by := bytes.NewBuffer(buf[index : index+4])
								binary.Read(by, binary.LittleEndian, &v)
								arryData = append(arryData, int(v))
								index += 4
							case binaryValue:
								index++
								var len int32
								by := bytes.NewBuffer(buf[index : index+4])
								binary.Read(by, binary.LittleEndian, &len)
								index += 4
								bin := buf[index : index+int(len)]
								arryData = append(arryData, bin)
								index += int(len)
							default:
								return nil, 0, errors.New("get single array type need to repaire")
							}
						}
						data[name] = arryData
						index++
					}
				}
			case binaryValue:
				index++
				var len int32
				by := bytes.NewBuffer(buf[index : index+4])
				binary.Read(by, binary.LittleEndian, &len)
				index += 4
				bin := buf[index : index+int(len)]
				data[name] = bin
				index += int(len)
			default:
				return nil, 0, errors.New("value error")
			}
		} else if buf[index] == stringValue {
			index++
			strCount := 0
			startIndex := index
			for buf[index] != 0 {
				strCount++
				index++
			}
			by := bytes.NewBuffer(buf[startIndex : startIndex+strCount])
			name := by.String()
			index++
			if buf[index] == arrayValue {
				index++
				arryData := []interface{}{}
				for buf[index] != endValue {
					switch buf[index] {
					case stringValue:
						index++
						strCount := 0
						startIndex := index
						for buf[index] != 0 {
							strCount++
							index++
						}
						by := bytes.NewBuffer(buf[startIndex : startIndex+strCount])
						arryData = append(arryData, by.String())
						index++
					default:

					}
				}
				data[name] = arryData
				index++
			}
		} else {
			break
		}

	}
	if buf[index] != endValue {
		return nil, 0, errors.New("end error")
	}
	index++
	return data, index, nil
}
