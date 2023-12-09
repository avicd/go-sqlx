package builder

import (
	"github.com/avicd/go-sqlx/script"
	"github.com/avicd/go-utilx/conv"
	"github.com/avicd/go-utilx/xmlx"
	"strings"
)

func handleTrim(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	prefix := node.AttrString("prefix")
	prefixOverrides := conv.StrToArr(node.AttrString("prefixOverrides"), "|")
	suffix := node.AttrString("suffix")
	suffixOverrides := conv.StrToArr(node.AttrString("suffixOverrides"), "|")
	return &script.TrimNode{
		Content:            content,
		Prefix:             prefix,
		PrefixesToOverride: prefixOverrides,
		Suffix:             suffix,
		SuffixesToOverride: suffixOverrides,
	}
}

func handleWhere(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	return &script.TrimNode{
		Content:            content,
		Prefix:             "WHERE",
		PrefixesToOverride: []string{"AND ", "OR ", "AND\n", "OR\n", "AND\r", "OR\r", "AND\t", "OR\t"},
	}
}

func handleSet(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	return &script.TrimNode{
		Content:            content,
		Prefix:             "SET",
		PrefixesToOverride: []string{","},
		SuffixesToOverride: []string{","},
	}
}

func handleForeach(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	collection := node.AttrString("collection")
	item := node.AttrString("item")
	index := node.AttrString("index")
	open := node.AttrString("open")
	clos := node.AttrString("close")
	separator := node.AttrString("separator")
	return &script.ForeachNode{
		Content:    content,
		Collection: collection,
		Item:       item,
		Index:      index,
		Open:       open,
		Close:      clos,
		Separator:  separator,
	}
}

func handleIf(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	test := node.AttrString("test")
	return &script.IfNode{Content: content, Test: test}
}

func handleOtherwise(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	return content
}

func handleChoose(node *xmlx.Node) script.SqlNode {
	var ifNodes []script.SqlNode
	var defaultIfNode script.SqlNode
	for p := node.FirstChild; p != nil; p = p.NextSibling {
		if p.Name == "if" {
			ifNodes = append(ifNodes, handleIf(p))
		}
		if p.Name == "otherwise" {
			defaultIfNode = handleOtherwise(p)
		}
	}
	return &script.ChooseNode{IfNodes: ifNodes, Default: defaultIfNode}
}

func handleBind(node *xmlx.Node) script.SqlNode {
	name := node.AttrString("name")
	value := node.AttrString("value")
	return &script.BindNode{Name: name, Value: value}
}

func handleInclude(node *xmlx.Node) script.SqlNode {
	content, _ := ParseSqlNode(node)
	return &script.IncludeNode{Content: content}
}

func ParseSqlNode(rootNode *xmlx.Node) (script.SqlNode, bool) {
	var sqlNodes []script.SqlNode
	isDynamic := false
	for _, p := range rootNode.ChildNodes {
		if p.Type == xmlx.CDataSectionNode || p.Type == xmlx.TextNode {
			text := strings.TrimSpace(p.Value)
			if text == "" {
				continue
			}
			node := &script.TextNode{Text: text}
			if node.IsDynamic() {
				isDynamic = true
				sqlNodes = append(sqlNodes, node)
			} else {
				sqlNodes = append(sqlNodes, &script.StaticNode{Text: text})
			}
		} else if p.Type == xmlx.ElementNode {
			var next script.SqlNode
			switch p.Name {
			case "trim":
				next = handleTrim(p)
			case "where":
				next = handleWhere(p)
			case "set":
				next = handleSet(p)
			case "foreach":
				next = handleForeach(p)
			case "if", "when":
				next = handleIf(p)
			case "otherwise":
				next = handleOtherwise(p)
			case "choose":
				next = handleChoose(p)
			case "binding":
				next = handleBind(p)
			case "include":
				next = handleInclude(p)
			default:
				continue
			}
			sqlNodes = append(sqlNodes, next)
			isDynamic = true
		}
	}
	return &script.ProxyNode{SqlNodes: sqlNodes}, isDynamic
}
