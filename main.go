package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

//Allowed flags
const FLAG_ID string = "id"
const FLAG_OPERATION string = "operation"
const FLAG_ITEM string = "item"
const FLAG_FILENAME string = "fileName"

//Allowed operation types
const OPERATION_ADD string = "add"
const OPERATION_LIST string = "list"
const OPERATION_FIND string = "findById"
const OPERATION_RM string = "remove"

//Allowed file extensions
const FILE_EXT string = "json"

//default file permissions (-rw -r -r)
const FILE_PERMISSIONS = 0644

//describes meanings of flags
const describe_operation string = "describes the type of operation on the received item." +
	"Allowed values: \n add: add new item in file;\n list: returns list of items in file;" +
	"\n findById: returns item from list by id;\n remove: delete item from list by id"
const describe_id string = "means id of item in file. Use this flag only with finndById and remove operations"
const describe_item string = "json-object which describes user by id, email and age fields"
const describe_filename string = "name of file which contains list of users (items). Only .json allowed"

var errorBadOperationType error = errors.New("Operation %s not allowed!")
var errorEmptyId error = errors.New("-id flag has to be specified")
var errorEmptyOperation error = errors.New("-operation flag has to be specified")
var errorEmptyItem error = errors.New("-item flag has to be specified")
var errorEmptyFileName error = errors.New("-fileName flag has to be specified")
var errorInvalidFileExtension error = errors.New("bad file extension. Use -h to see aloowed extensions")

//Input map of flags value from cmdl
type Arguments map[string]string

type User struct {
	Id    string `json:"id"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}
type UserList []User

//
func checkFilename(filename string) error {
	if filename == "" {
		return errorEmptyFileName
	}

	extension := strings.Split(filename, ".")
	if extension[len(extension)-1] != FILE_EXT {
		return errorInvalidFileExtension
	}

	return nil
}

//check flags for allowed values
func checkFlags(args Arguments) error {

	var (
		id, item, operation, filename string
	)
	id = args[FLAG_ID]
	item = args[FLAG_ITEM]
	operation = args[FLAG_OPERATION]
	filename = args[FLAG_FILENAME]

	if operation == "" {
		return errorEmptyOperation
	}

	if operation != OPERATION_ADD && operation != OPERATION_FIND && operation != OPERATION_LIST && operation != OPERATION_RM {
		return fmt.Errorf(errorBadOperationType.Error(), operation)
	}

	if operation == OPERATION_ADD {
		if item == "" || item == "{}" {
			return errorEmptyItem
		}
	}

	if operation == OPERATION_FIND || operation == OPERATION_RM {
		if id == "" {
			return errorEmptyId
		}
	}

	err := checkFilename(filename)
	if err != nil {
		return err
	}

	return nil
}

//read flags from cmd
func parseArgs() (Arguments, error) {
	var (
		argv                          Arguments
		id, operation, item, filename string
	)

	flag.StringVar(&id, FLAG_ID, "", describe_id)
	flag.StringVar(&operation, FLAG_OPERATION, "", describe_operation)
	flag.StringVar(&item, FLAG_ITEM, "", describe_item)
	flag.StringVar(&filename, FLAG_FILENAME, "", describe_filename)

	flag.Parse()

	//init map
	argv = make(Arguments)
	argv[FLAG_ID] = id
	argv[FLAG_OPERATION] = operation
	argv[FLAG_ITEM] = item
	argv[FLAG_FILENAME] = filename

	return argv, nil
}

//add passed item into passed file
func addUser(item string, file *os.File) ([]byte, error) {
	var (
		buffer    []byte
		nUser     User
		nUserList UserList
		err       error
	)

	buffer, err = io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(item), &nUser)
	if err != nil {
		return nil, err
	}

	if len(buffer) > 0 {
		//if we found some data in file
		err = json.Unmarshal(buffer, &nUserList)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(nUserList); i++ {
			if nUserList[i].Id == nUser.Id {
				buffer = []byte("Item with id " + nUser.Id + " already exists")
				return buffer, nil
			}
		}
	}

	nUserList = append(nUserList, nUser)
	buffer, err = json.Marshal(&nUserList)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(file.Name(), buffer, FILE_PERMISSIONS)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

//returns item that has passed id
func findUserById(id string, file *os.File) ([]byte, error) {
	var (
		buffer, result []byte
		vList          UserList
		err            error
	)

	buffer, err = io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(buffer, &vList)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(vList); i++ {
		if vList[i].Id == id {
			result, err := json.Marshal(&vList[i])
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}
	return result, nil
}

func removeUserById(id string, file *os.File) ([]byte, error) {
	var (
		usList       UserList
		buffer, data []byte
		err          error
	)
	buffer, err = io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if len(buffer) > 0 {
		//if we found some data in file
		err = json.Unmarshal(buffer, &usList)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(usList); i++ {
			if usList[i].Id == id {
				//delete element
				usList[i] = usList[len(usList)-1]
				usList[len(usList)-1] = User{}
				usList = usList[:len(usList)-1]

				//create new data
				data, err := json.Marshal(&usList)
				if err != nil {
					return nil, err
				}
				err = ioutil.WriteFile(file.Name(), data, FILE_PERMISSIONS)
				if err != nil {
					return nil, err
				}
				return nil, nil
			}
		}
	}
	data = []byte("Item with id " + id + " not found")
	return data, nil
}

//listing of users in file
func listUsers(file *os.File) ([]byte, error) {
	buff, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return buff, nil
}

func Perform(args Arguments, writer io.Writer) error {
	var (
		err  error
		file *os.File
		data []byte
	)

	//check flags
	err = checkFlags(args)
	if err != nil {
		return err
	}

	//open file is exists, or create if doesn`t
	file, err = os.OpenFile(args["fileName"], os.O_RDWR|os.O_CREATE, FILE_PERMISSIONS)
	defer file.Close()

	if err != nil {
		return err
	}

	switch args["operation"] {
	case OPERATION_ADD:
		data, err = addUser(args["item"], file)
		if err != nil {
			return err
		}
		writer.Write(data)
	case OPERATION_LIST:
		data, err = listUsers(file)
		if err != nil {
			return err
		}
		writer.Write(data)
	case OPERATION_FIND:
		data, err = findUserById(args["id"], file)
		if err != nil {
			return err
		}
		writer.Write(data)
	case OPERATION_RM:
		data, err = removeUserById(args["id"], file)
		if err != nil {
			return err
		}
		writer.Write(data)
	default:
		return fmt.Errorf(errorBadOperationType.Error(), args["operation"])
	}

	return nil
}

func main() {
	var err error
	args, err := parseArgs()
	if err != nil {
		panic(err)
	}

	err = Perform(args, os.Stdout)
	if err != nil {
		panic(err)
	}
}
