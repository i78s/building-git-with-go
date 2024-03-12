package config

import (
	"bufio"
	"building-git/lib/lockfile"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	SECTION_LINE  = regexp.MustCompile(`(?i)^\s*\[([a-z0-9-]+)( "(.+)")?\]\s*($|#|;)`)
	VARIABLE_LINE = regexp.MustCompile(`(?mi)^\s*([a-z][a-z0-9-]*)\s*=\s*(.*?)\s*($|#|;)`)
	BLANK_LINE    = regexp.MustCompile(`^\s*($|#|;)`)
	INTEGER       = regexp.MustCompile(`^-?[1-9][0-9]*$`)
)

type ConflictError struct {
	message string
}

func (c ConflictError) Error() string {
	return c.message
}

type ParseError struct {
	message string
}

func (p ParseError) Error() string {
	return p.message
}

type Line struct {
	text     string
	section  *Section
	variable *Variable
}

func (l *Line) NormalVariable() string {
	if l.variable != nil {
		return NormalizeVariable(l.variable.name)
	}
	return ""
}

type Section struct {
	name []string
}

func NormalizeSection(name []string) string {
	if len(name) == 0 {
		return ""
	}
	return strings.ToLower(name[0]) + strings.Join(name[1:], ".")
}

func (s *Section) HeadingLine() string {
	line := fmt.Sprintf("[%s", s.name[0])
	if len(s.name) > 1 {
		line += fmt.Sprintf(" \"%s\"", strings.Join(s.name[1:], "."))
	}
	line += "]\n"
	return line
}

type Variable struct {
	name  string
	value interface{}
}

func NormalizeVariable(name string) string {
	return strings.ToLower(name)
}

func SerializeVariable(name string, value interface{}) string {
	return fmt.Sprintf("\t%s = %v\n", name, value)
}

type Config struct {
	path     string
	lockfile *lockfile.Lockfile
	lines    map[string][]*Line
	lineKeys []string
}

func NewConfig(path string) *Config {
	return &Config{
		path:     path,
		lockfile: lockfile.NewLockfile(path),
		lines:    nil,
	}
}

func (c *Config) Open() error {
	if c.lines == nil {
		err := c.readConfigFile()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) OpenForUpdate() error {
	err := c.lockfile.HoldForUpdate()
	if err != nil {
		return err
	}
	return c.readConfigFile()
}

func (c *Config) Save() error {
	for _, key := range c.lineKeys {
		for _, line := range c.lines[key] {
			err := c.lockfile.Write([]byte(line.text))
			if err != nil {
				return err
			}
		}
	}
	return c.lockfile.Commit()
}

func (c *Config) Get(key []string) (interface{}, error) {
	values, err := c.GetAll(key)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values[len(values)-1], nil
}

func (c *Config) GetAll(key []string) ([]interface{}, error) {
	k, v := splitKey(key)
	_, lines, err := c.findLines(k, v)
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, line := range lines {
		values = append(values, line.variable.value)
	}
	return values, nil
}

func (c *Config) Add(key []string, value interface{}) {
	k, v := splitKey(key)
	section, _, _ := c.findLines(k, v)

	c.addVariable(section, k, v, value)
}

func (c *Config) Set(key []string, value interface{}) error {
	k, v := splitKey(key)
	section, lines, err := c.findLines(k, v)
	if err != nil {
		return err
	}

	switch len(lines) {
	case 0:
		c.addVariable(section, k, v, value)
	case 1:
		lines[0].variable.value = value
		lines[0].text = SerializeVariable(v, value)
	default:
		return &ConflictError{message: "cannot overwrite multiple values with a single value"}
	}
	return nil
}

func (c *Config) ReplaceAll(key []string, value interface{}) {
	k, v := splitKey(key)
	section, lines, err := c.findLines(k, v)
	if err != nil {
		return
	}

	c.removeAll(section, lines)
	c.addVariable(section, k, v, value)
}

func splitKey(key []string) ([]string, string) {
	k := key[:len(key)-1]
	v := key[len(key)-1:][0]
	return k, v
}

func (c *Config) findLines(keyName []string, varName string) (*Section, []*Line, error) {
	name := NormalizeSection(keyName)
	lines, ok := c.lines[name]
	if !ok {
		return nil, nil, nil
	}
	section := lines[0].section
	noraml := NormalizeVariable(varName)

	var matchingLines []*Line
	for _, line := range lines {
		if line.variable != nil && strings.EqualFold(line.variable.name, noraml) {
			matchingLines = append(matchingLines, line)
		}
	}

	return section, matchingLines, nil
}

func (c *Config) addSection(keyName []string) *Section {
	section := &Section{name: keyName}
	line := &Line{
		text:    section.HeadingLine(),
		section: section,
	}
	key := NormalizeSection(section.name)
	c.lineKeys = append(c.lineKeys, key)
	c.lines[key] = append(c.lines[key], line)
	return section
}

func (c *Config) addVariable(section *Section, keyName []string, varName string, value interface{}) {
	if section == nil {
		section = c.addSection(keyName)
	}

	text := SerializeVariable(varName, value)
	variable := &Variable{name: varName, value: value}
	line := &Line{
		text:     text,
		section:  section,
		variable: variable,
	}
	c.lines[NormalizeSection(section.name)] = append(c.lines[NormalizeSection(section.name)], line)
}

func (c *Config) removeAll(section *Section, lines []*Line) {
	for _, line := range lines {
		sectionLines := c.lines[NormalizeSection(section.name)]
		newLines := []*Line{}
		for _, l := range sectionLines {
			if l != line {
				newLines = append(newLines, l)
			}
		}
		c.lines[NormalizeSection(section.name)] = newLines
	}
}

func (c *Config) readConfigFile() error {
	c.lines = map[string][]*Line{}
	c.lineKeys = []string{}
	section := &Section{}

	file, err := os.Open(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	lineNum := 0

	for {
		line, err := readLine(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		lineNum++
		parsedLine, err := c.parseLine(section, line, lineNum)
		if err != nil {
			return err
		}

		section = parsedLine.section
		key := NormalizeSection(section.name)
		if len(c.lines[key]) == 0 {
			c.lineKeys = append(c.lineKeys, key)
		}
		c.lines[key] = append(c.lines[key], parsedLine)
	}

	return nil
}

func readLine(reader *bufio.Reader) (string, error) {
	var line strings.Builder
	for {
		tmp, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if line.Len() > 0 {
					return line.String(), nil
				}
				return "", io.EOF
			}
			return "", err
		}

		line.WriteString(tmp)
		if !strings.HasSuffix(tmp, "\\\n") {
			break
		}
	}
	return line.String(), nil
}

func (c *Config) parseLine(section *Section, line string, lineNum int) (*Line, error) {
	if match := SECTION_LINE.FindStringSubmatch(line); match != nil {
		sectionName := match[1]
		sectionComment := match[3]
		section = &Section{name: []string{sectionName, sectionComment}}
		return &Line{text: line, section: section}, nil
	} else if match := VARIABLE_LINE.FindStringSubmatch(line); match != nil {
		varName := match[1]
		varValue := parseValue(match[2])
		variable := &Variable{name: varName, value: varValue}
		return &Line{text: line, section: section, variable: variable}, nil
	} else if BLANK_LINE.MatchString(line) {
		return &Line{text: line, section: section}, nil
	}

	return nil, ParseError{message: fmt.Sprintf("bad config line %d in file %s", lineNum, c.path)}
}

func parseValue(value string) interface{} {
	switch strings.ToLower(value) {
	case "yes", "on", "true":
		return true
	case "no", "off", "false":
		return false
	}

	if INTEGER.MatchString(value) {
		intValue, _ := strconv.Atoi(value)
		return intValue
	}

	return strings.ReplaceAll(value, "\\\n", "")
}
