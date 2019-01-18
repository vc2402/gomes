package schema

import (
	"io/ioutil"

	"github.com/spf13/viper"
)

func GetSchema() (string, error) {
	path := viper.GetString("SchemaPath")
	if path == "" {
		path = "./schema/schema.graphql"
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
