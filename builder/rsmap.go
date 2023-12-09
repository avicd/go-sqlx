package builder

import (
	"github.com/avicd/go-sqlx/logger"
	"github.com/avicd/go-sqlx/session"
	"github.com/avicd/go-utilx/conv"
	"github.com/avicd/go-utilx/refx"
	"github.com/avicd/go-utilx/xmlx"
)

type RsMapBuilder struct {
	cfg    *session.Config
	Ns     string
	Nodes  []*xmlx.Node
	Linked map[string]bool
	RsMaps map[string]session.RsMap
	ExIds  map[string]string
}

func (it *RsMapBuilder) Build(config *session.Config) {
	it.cfg = config
	it.RsMaps = make(map[string]session.RsMap)
	it.ExIds = make(map[string]string)
	it.Linked = make(map[string]bool)
	for _, rsNode := range it.Nodes {
		rsId := rsNode.AttrString("id")
		if rsId == "" {
			logger.Warnf("missing id <resultMap id=?, skipped [%s]", it.cfg.GetNsXml(it.Ns))
			continue
		}
		if it.RsMaps[rsId] != nil {
			logger.Errorf("duplicate id <resultMap id='%s',skipped [%s]", rsId, it.cfg.GetNsXml(it.Ns))
			continue
		}
		exId := rsNode.AttrString("extends")
		if exId != "" {
			it.ExIds[rsId] = exId
		}
		rsMap := session.RsMap{}
		results := rsNode.Find("result")
		for _, result := range results {
			column := result.AttrString("column")
			property := result.AttrString("property")
			if column == "" {
				logger.Warnf("missing column <resultMap id='%s'><result column=?, skipped [%s]", rsId, it.cfg.GetNsXml(it.Ns))
				continue
			}
			if property == "" {
				logger.Warnf("missing column <resultMap id='%s'><result property=?, skipped [%s]", rsId, it.cfg.GetNsXml(it.Ns))
				continue
			}
			rsMap[column] = conv.StrToArr(property, ".")
		}
		it.RsMaps[rsId] = rsMap
	}
	for rsId := range it.ExIds {
		it.LinkExtend(rsId, nil)
	}
	for rsId, rsMap := range it.RsMaps {
		if len(rsMap) < 1 {
			continue
		}
		it.cfg.AddRsMap(it.Ns+"."+rsId, rsMap)
	}
}

func (it *RsMapBuilder) LinkExtend(rsId string, idStack map[string]bool) {
	if it.Linked[rsId] {
		return
	}
	exIds := it.ExIds[rsId]
	if exIds == "" {
		it.Linked[rsId] = true
		return
	}
	var newIdStack map[string]bool
	refx.Clone(&newIdStack, idStack)
	newIdStack[rsId] = true
	for _, exId := range conv.StrToArr(exIds, ",") {
		if it.RsMaps[exId] == nil {
			logger.Fatalf("missing resultMap '%s' at linking extend [%s]", exId, it.cfg.GetNsXml(it.Ns))
		}
		if newIdStack[exId] {
			logger.Fatalf("circular extending resultMap id '%s' [%s]", exId, it.cfg.GetNsXml(it.Ns))
		}
		if !it.Linked[exId] {
			it.LinkExtend(exId, newIdStack)
		}
		var newRsMap session.RsMap
		refx.Clone(&newRsMap, it.RsMaps[exId])
		refx.Merge(newRsMap, it.RsMaps[rsId])
		it.RsMaps[rsId] = newRsMap
	}
	it.Linked[rsId] = true
}
