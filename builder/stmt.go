package builder

import (
	"fmt"
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/script"
	"github.com/avicd/go-sqlx/session"
	"github.com/avicd/go-utilx/conv"
	"github.com/avicd/go-utilx/evalx"
	"github.com/avicd/go-utilx/refx"
	"github.com/avicd/go-utilx/tokx"
	"github.com/avicd/go-utilx/xmlx"
	"regexp"
	"strings"
)

var cmdTypes = map[string]session.StmtType{
	"select":    session.Select,
	"selectSet": session.SelectSet,
	"insert":    session.Insert,
	"update":    session.Update,
	"delete":    session.Delete,
}

var setTargets = map[string]session.Target{
	"ARG":     session.ARG,
	"RESULT":  session.RESULT,
	"CONTEXT": session.CONTEXT,
}

type StmtBuilder struct {
	cfg      *session.Config
	Ns       string
	RefSql   map[string]*xmlx.Node
	Nodes    []*xmlx.Node
	Stmts    map[string]*session.Stmt
	SetNodes map[string][]*xmlx.Node
}

func (it *StmtBuilder) Build(config *session.Config) {
	it.cfg = config
	it.Stmts = make(map[string]*session.Stmt)
	it.SetNodes = map[string][]*xmlx.Node{}
	for _, stNode := range it.Nodes {
		it.ParseStmt(stNode, nil)
	}
	for stId, setNodes := range it.SetNodes {
		if setNodes == nil {
			continue
		}
		for _, setNode := range setNodes {
			it.ParseStmt(setNode, it.Stmts[stId])
		}
	}
}

func (it *StmtBuilder) LinkInclude(node *xmlx.Node, idStack map[string]bool, pStack map[string]string) {
	ref := node.AttrString("ref")
	target := it.RefSql[ref]
	if target == nil {
		logger.Fatalf("missing sql <sql id='%s' at including [%s]", ref, it.cfg.GetNsXml(it.Ns))
	}

	newIdStack := map[string]bool{}
	refx.Merge(&newIdStack, idStack)
	newIdStack[ref] = true
	newPStack := map[string]string{}
	refx.Merge(&newPStack, pStack)
	props := node.Find("property")
	for _, prop := range props {
		name := prop.AttrString("name")
		value := prop.AttrString("value")
		newPStack[name] = value
	}
	xmlText := target.InnerXML()
	scope := evalx.NewScope(newPStack)
	xmlText = tokx.NewPair("${", "}").Map(xmlText, func(s string) string {
		val, _ := scope.Eval(s)
		result := refx.AsString(val)
		if result != "" && regexp.MustCompile(script.InjectPattern).MatchString(result) {
			logger.Fatalf("invalid value <property value='%s' at linking include [%s] \nplease conform to regex \"%s\"", result, it.cfg.GetNsXml(it.Ns), script.InjectPattern)
		}
		return result
	})
	newNode, _ := xmlx.Parse(strings.NewReader(xmlText))
	if newNode != nil {
		node.ClearContent()
		node.AppendAll(newNode.ChildNodes)
	}
	if newNode.HasChildren() {
		includes := newNode.Find("//include")
		for _, nd := range includes {
			ref = nd.AttrString("ref")
			if newIdStack[ref] {
				logger.Fatalf("circular including <sql id='%s' [%s]", ref, it.cfg.GetNsXml(it.Ns))
			}
			it.LinkInclude(nd, newIdStack, newPStack)
		}
	}
}

func (it *StmtBuilder) GetRsMaps(node *xmlx.Node) []session.RsMap {
	mapIds := conv.StrToArr(node.AttrString("resultMap"), ",")
	var rsMaps []session.RsMap
	for _, mapId := range mapIds {
		rsMap := it.cfg.GetRsMap(it.Ns + "." + mapId)
		if rsMap == nil {
			logger.Fatalf("missing resultMap <resultMap id='%s' at parsing statement [%s]", mapId, it.cfg.GetNsXml(it.Ns))
		}
		rsMaps = append(rsMaps, rsMap)
	}
	return rsMaps
}

func (it *StmtBuilder) ParseStmt(node *xmlx.Node, parent *session.Stmt) *session.Stmt {
	stId := node.AttrString("id")
	if stId == "" {
		logger.Fatalf("missing id <%s id=?", node.Name)
	}
	var id string
	isSelectSet := node.Name == "selectSet"
	if isSelectSet {
		id = fmt.Sprintf("%s%s.%s", parent.Id, session.SelectSetSuffix, stId)
	} else {
		includes := node.Find("include")
		for _, include := range includes {
			it.LinkInclude(include, nil, nil)
		}
		id = it.Ns + "." + stId
	}
	cmdType := cmdTypes[node.Name]
	argNames := conv.StrToArr(node.AttrString("args"), ",")
	keyProps := conv.StrToArr(node.AttrString("keyProp"), ",")
	keyColumns := conv.StrToArr(node.AttrString("keyColumn"), ",")
	dbId := node.AttrString("databaseId")
	bind := node.AttrString("bind")
	rsMaps := it.GetRsMaps(node)
	it.SetNodes[stId] = node.Find("selectSet")
	var sqlNode script.SqlNode
	var dynamic bool
	var setTarget session.Target
	if isSelectSet {
		target := node.AttrString("target")
		setTarget = setTargets[target]
		if len(keyProps) < 1 && len(parent.KeyProps) > 0 {
			keyProps = parent.KeyProps
			keyColumns = parent.KeyColumns
		}
		if len(rsMaps) < 0 && len(keyProps) > 0 && len(keyProps) == len(keyColumns) {
			rsMap := session.RsMap{}
			for i, k := range keyColumns {
				rsMap[k] = conv.StrToArr(keyProps[i], ".")
			}
			rsMaps = []session.RsMap{rsMap}
		}
		ref := node.AttrString("ref")
		if ref != "" {
			refStmt := it.Stmts[ref]
			if refStmt == nil {
				logger.Fatalf("missing statement <select id='%s' at linking selectSet ref [%s]", ref, it.cfg.GetNsXml(it.Ns))
			} else if refStmt.StmtType != session.Select {
				logger.Fatalf("statement '%s' is not select statement", refStmt.Id)
			}
			if len(argNames) < 1 {
				argNames = refStmt.ArgNames
			}
			sqlNode = refStmt.SqlNode
			dynamic = refStmt.Dynamic
		} else {
			sqlNode, dynamic = ParseSqlNode(node)
		}
	} else {
		sqlNode, dynamic = ParseSqlNode(node)
	}
	stmt := &session.Stmt{
		Id:         id,
		Ns:         it.Ns,
		DbId:       dbId,
		SqlNode:    sqlNode,
		Dynamic:    dynamic,
		RsMaps:     rsMaps,
		StmtType:   cmdType,
		ArgNames:   argNames,
		KeyProps:   keyProps,
		KeyColumns: keyColumns,
		Target:     setTarget,
		Bind:       bind,
	}
	it.cfg.AddStmt(stmt)
	it.Stmts[stId] = stmt
	if isSelectSet {
		parent.SetStmts = append(parent.SetStmts, stmt)
	}
	return stmt
}
