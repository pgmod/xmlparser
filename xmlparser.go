package xmlparser

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// Структура для хранения тега XML
type XMLTag struct {
	Name       string            // Имя тега
	Attributes map[string]string // Атрибуты тега
	Content    string            // Текстовый контент внутри тега
	Children   []XMLTag          // Спаршенные дочерние теги
}

func (tag *XMLTag) Find(pattern string) XMLTag {
	res, _ := searchInTag(tag, pattern)

	return res
}

func searchInTag(rootTag *XMLTag, pattern string) (XMLTag, error) {

	// Если парсинг не удался, возвращаем ошибку
	if rootTag.Name == "" {
		return XMLTag{}, fmt.Errorf("ошибка парсинга XML")
	}

	// Разбираем паттерн
	patternParts := parsePattern(pattern)

	// Начинаем поиск по паттерну
	result, err := findTagByPattern(*rootTag, patternParts)
	if err != nil {
		return XMLTag{}, err
	}

	return result, nil
}

// searchInXML - Основная функция для поиска по паттерну в XML
// Принимает XML строку и паттерн, возвращает результат поиска или ошибку
// func searchInXML(xmlData string, pattern string) XMLTag {
// 	// Парсим XML строку в структуру
// 	rootTag := ParseXML(xmlData)
// 	return rootTag.Find(pattern)

// }

// parsePattern - Функция для разбиения паттерна на составляющие
// Возвращает массив строк, которые представляют путь поиска
func parsePattern(pattern string) []string {
	// Убираем ведущий и завершающий слеши
	trimmedPattern := strings.Trim(pattern, "/")

	// Разбиваем паттерн на части по символу "/"
	patternParts := strings.Split(trimmedPattern, "/")

	return patternParts
}

// findTagByPattern - Рекурсивная функция для поиска по XML структуре согласно паттерну
// Принимает XMLTag и массив частей паттерна, возвращает найденный XMLTag или ошибку
func findTagByPattern(tag XMLTag, patternParts []string) (XMLTag, error) {
	// Если паттерн пустой, возвращаем текущий тег
	if len(patternParts) == 0 {
		return tag, nil
	}

	// Берем текущий шаг паттерна
	currentPattern := patternParts[0]

	// Проверяем, является ли текущий шаг атрибутом (начинается с @)
	if strings.HasPrefix(currentPattern, "@") {
		attributeName := strings.TrimPrefix(currentPattern, "@")
		// Ищем атрибут в текущем теге
		if value, ok := tag.Attributes[attributeName]; ok {
			// Возвращаем тег с заполненным полем Content
			return XMLTag{
				Content: value,
			}, nil
		}
		return XMLTag{}, fmt.Errorf("атрибут %s не найден", attributeName)
	}

	// Проверяем, является ли текущий шаг индексом (например, [0])
	if strings.HasPrefix(currentPattern, "[") && strings.HasSuffix(currentPattern, "]") {
		indexString := strings.Trim(currentPattern, "[]")
		index, err := parseIndex(indexString)
		if err != nil {
			return XMLTag{}, err
		}

		// Проверяем, есть ли дочерний тег с таким индексом
		if index >= 0 && index < len(tag.Children) {
			// Продолжаем рекурсивный поиск в дочернем теге
			return findTagByPattern(tag.Children[index], patternParts[1:])
		}
		return XMLTag{}, fmt.Errorf("тег с индексом %d не найден", index)
	}

	// Ищем тег с соответствующим именем среди дочерних тегов
	for _, child := range tag.Children {
		if child.Name == currentPattern {
			// Продолжаем рекурсивный поиск в найденном дочернем теге
			return findTagByPattern(child, patternParts[1:])
		}
	}

	return XMLTag{}, fmt.Errorf("тег %s не найден", currentPattern)
}

// parseIndex - Вспомогательная функция для парсинга индекса из строки
func parseIndex(indexString string) (int, error) {
	// Преобразуем строку в число
	index, err := strconv.Atoi(indexString)
	if err != nil {
		return 0, fmt.Errorf("неверный формат индекса: %s", indexString)
	}
	return index, nil
}

// ParseXML парсит переданную строку XML и возвращает структуру для корневого тега
func ParseXML(xmlString string) XMLTag {
	decoder := xml.NewDecoder(strings.NewReader(xmlString)) // Создаем XML-декодер из строки
	var rootTag XMLTag
	var contentBuilder strings.Builder // Используем Builder для построения контента

	// Цикл по элементам XML
	for {
		// Читаем следующий токен
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break // Конец документа
			}
			return XMLTag{}
		}

		// Обрабатываем тип токена
		switch elem := token.(type) {
		case xml.StartElement:
			// Если это первый элемент, считаем его корневым
			if rootTag.Name == "" {
				rootTag.Name = elem.Name.Local
				rootTag.Attributes = make(map[string]string)

				// Добавляем атрибуты в карту
				for _, attr := range elem.Attr {
					rootTag.Attributes[attr.Name.Local] = attr.Value
				}
			} else {
				// Парсим вложенный элемент рекурсивно
				childTag, err := parseXMLElement(decoder, elem)
				if err != nil {
					return XMLTag{}
				}
				rootTag.Children = append(rootTag.Children, childTag)

				// Сохраняем контент вложенных тегов
				contentBuilder.WriteString(serializeStartElement(elem))
				contentBuilder.WriteString(childTag.Content)
				contentBuilder.WriteString("</" + elem.Name.Local + ">")
			}

		case xml.CharData:
			// Добавляем контент как строку
			content := strings.TrimSpace(string(elem))
			if len(content) > 0 {
				contentBuilder.WriteString(content)
			}
		}
	}

	// Присваиваем весь контент корневому тегу
	rootTag.Content = contentBuilder.String()

	return rootTag
}

// parseXMLElement парсит вложенные элементы рекурсивно
func parseXMLElement(decoder *xml.Decoder, startElement xml.StartElement) (XMLTag, error) {
	var tag XMLTag
	tag.Name = startElement.Name.Local
	tag.Attributes = make(map[string]string)
	var contentBuilder strings.Builder

	// Добавляем атрибуты в карту
	for _, attr := range startElement.Attr {
		tag.Attributes[attr.Name.Local] = attr.Value
	}

	// Цикл для обработки вложенных элементов
	for {
		token, err := decoder.Token()
		if err != nil {
			return XMLTag{}, err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Рекурсивный вызов для обработки вложенных элементов
			childTag, err := parseXMLElement(decoder, elem)
			if err != nil {
				return XMLTag{}, err
			}
			tag.Children = append(tag.Children, childTag)

		case xml.CharData:
			// Добавляем только текстовый контент в текущий тег
			content := strings.TrimSpace(string(elem))
			if len(content) > 0 {
				contentBuilder.WriteString(content)
			}

		case xml.EndElement:
			// Завершаем обработку текущего тега
			if elem.Name.Local == tag.Name {
				tag.Content = contentBuilder.String()
				return tag, nil
			}
		}
	}
}

// Вспомогательная функция для сериализации тега
func serializeStartElement(elem xml.StartElement) string {
	var result strings.Builder
	result.WriteString("<" + elem.Name.Local)
	for _, attr := range elem.Attr {
		result.WriteString(" " + attr.Name.Local + "=\"" + attr.Value + "\"")
	}
	result.WriteString(">")
	return result.String()
}
