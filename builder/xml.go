package builder

import (
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/session"
	"github.com/avicd/go-utilx/xmlx"
	"github.com/bmatcuk/doublestar/v4"
	"os"
	"strings"
)

type XmlBuilder struct {
	cfg  *session.Config
	Scan string
}

func (it *XmlBuilder) Build(config *session.Config) {
	it.cfg = config
	if it.Scan == "" {
		it.Scan = it.cfg.XmlScan
	}
	pattern := strings.TrimSpace(it.Scan)
	if pattern == "" {
		pattern = "**/*.xml"
	}
	xmlFiles, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	for _, xmlFile := range xmlFiles {
		bytes, err := os.ReadFile(xmlFile)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		doc, err := xmlx.Parse(strings.NewReader(string(bytes)))
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		mapper := doc.FindOne("/mapper")
		if mapper == nil {
			continue
		}
		var ns string
		if ns = mapper.AttrString("namespace"); ns == "" {
			logger.Warnf("missing namespace <mapper namespace=?,skipped [%s]", xmlFile)
			continue
		}
		if it.cfg.HasNs(ns) {
			logger.Warnf("duplicate namespace <mapper namespace='%s', skipped [%s]", ns, xmlFile)
			continue
		}
		it.cfg.AddNs(ns, xmlFile)
		rsNodes := mapper.Find("resultMap")
		rsBuilder := &RsMapBuilder{Ns: ns, Nodes: rsNodes}
		rsBuilder.Build(it.cfg)

		sqNodes := mapper.Find("sql")
		refSql := map[string]*xmlx.Node{}
		for _, sqNode := range sqNodes {
			sqId := sqNode.AttrString("id")
			if sqId == "" {
				logger.Warnf("missing id <sql id=?, skipped [%s]", xmlFile)
				continue
			}
			refSql[sqId] = sqNode
		}
		stNodes := mapper.Find("select|insert|update|delete")
		stBuilder := &StmtBuilder{Ns: ns, Nodes: stNodes, RefSql: refSql}
		stBuilder.Build(it.cfg)
	}
}
