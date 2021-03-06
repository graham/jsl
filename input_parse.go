package jsl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"unicode"
)

func Nested_ReadJsonObjectsUntilEOF(objs chan interface{}, r io.Reader, failOnException bool) error {
	defer close(objs)

	var dec *json.Decoder = json.NewDecoder(r)
	var err error

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	handle_object := func() error {
		var json_obj interface{}
		err := dec.Decode(&json_obj)

		if err != nil && failOnException {
			return err
		}

		objs <- json_obj

		return nil
	}

	var handle_list func() error

	handle_list = func() error {
		token, err := dec.Token()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch token.(type) {
		case json.Delim:
			char := token.(json.Delim).String()
			if char[0] == '[' {
				for dec.More() {
					err = handle_object()

					if err != nil {
						return err
					}
				}
			} else {
				return errors.New("Nested input must start with a [")
			}
		default:
			return errors.New("Data is not nested in a list.")
		}
		return nil
	}

	return handle_list()
}

func Flatten_ReadJsonObjectsUntilEOF(objs chan interface{}, r io.Reader, failOnException bool) error {
	defer close(objs)

	var dec *json.Decoder = json.NewDecoder(r)
	var err error

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	var handle_list func() error

	handle_list = func() error {
		for {
			token, err := dec.Token()

			if err == io.EOF {
				return nil
			}

			if err != nil {
				return err
			}

			switch token.(type) {
			case json.Delim:
				char := token.(json.Delim).String()
				if char[0] == '[' {
					err = handle_list()

					if err != nil {
						return err
					}
				} else {
					return errors.New("Nested input must start with a [")
				}
			default:
				objs <- token
			}

			if dec.More() == false {
				return nil
			}
		}
	}

	return handle_list()
}

func ReadJsonObjectsUntilEOF(objs chan interface{}, r io.Reader, failOnException bool) error {
	defer close(objs)

	reader := bufio.NewReader(r)

	for {
		var readErr error

		line, readErr := reader.ReadString('\n')

		if readErr != nil && readErr != io.EOF {
			return readErr
		}

		line = strings.TrimSpace(line)

		if len(line) == 0 {
			if readErr == nil {
				continue
			} else if readErr == io.EOF {
				return nil
			} else {
				continue
			}
		}

		json_obj, err := LoadLine(line)

		if err != nil {
			log.Printf(fmt.Sprintf("json_decode_err: %s", err))
			if failOnException {
				return err
			}
		}

		if json_obj != nil {
			objs <- json_obj
		}

		if readErr == io.EOF {
			return nil
		}
	}

	return nil
}

func LoadLine(line string) (interface{}, error) {
	if line == "true" || line == "false" {
		var json_obj bool
		if err := json.Unmarshal([]byte(line), &json_obj); err != nil {
			return nil, err
		}
		return json_obj, nil
	} else if line[0] == '"' {
		var json_obj string
		if err := json.Unmarshal([]byte(line), &json_obj); err != nil {
			return nil, err
		}
		return json_obj, nil
	} else if line[0] == '[' {
		var json_obj []interface{}
		if err := json.Unmarshal([]byte(line), &json_obj); err != nil {
			return nil, err
		}
		return json_obj, nil
	} else if line[0] == '{' {
		var json_obj map[string]interface{} = make(map[string]interface{})
		if err := json.Unmarshal([]byte(line), &json_obj); err != nil {
			return nil, err
		}
		return json_obj, nil
	} else if unicode.IsDigit(rune(line[0])) {
		if strings.Contains(line, ".") {
			_, err := strconv.ParseFloat(line, 64)
			if err == nil {
				var json_obj float64
				if err := json.Unmarshal([]byte(line), &json_obj); err != nil {
					return nil, err
				}
				return json_obj, nil
			}
		} else {
			_, err := strconv.ParseInt(line, 10, 64)
			if err == nil {
				var json_obj int64
				if err := json.Unmarshal([]byte(line), &json_obj); err != nil {
					return nil, err
				}
				return json_obj, nil
			}
		}
	} else {
		log.Println("unknown type:", line)
		return nil, errors.New("Unknown type")
	}

	return nil, errors.New("Unknown type")
}
