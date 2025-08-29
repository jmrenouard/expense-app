package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "strings"

    "github.com/gin-gonic/gin"
)

// Translations maps language codes to key-value pairs.
var Translations = make(map[string]map[string]string)

// LoadTranslations reads translation files from a directory.
func LoadTranslations(dir string) error {
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        return err
    }
    for _, file := range files {
        if strings.HasSuffix(file.Name(), ".json") {
            lang := strings.TrimSuffix(file.Name(), ".json")
            content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
            if err != nil {
                return err
            }
            var translations map[string]string
            if err := json.Unmarshal(content, &translations); err != nil {
                return err
            }
            Translations[lang] = translations
        }
    }
    return nil
}

// GetMsg retrieves a translated string for a given key and context.
// It uses the Accept-Language header to determine the best language.
func GetMsg(c *gin.Context, key string) string {
    lang := c.GetHeader("Accept-Language")
    // Very basic language negotiation
    if t, ok := Translations[lang]; ok {
        if msg, ok := t[key]; ok {
            return msg
        }
    }
    // Fallback to English
    if t, ok := Translations["en"]; ok {
        if msg, ok := t[key]; ok {
            return msg
        }
    }
    return key // Return key if no translation found
}
