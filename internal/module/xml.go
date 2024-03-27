package module

import (
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func init() {
	register("xml", func(worker Worker, db Db) interface{} {
		return func(content string) (*XmlNode, error) {
			d, err := htmlquery.Parse(strings.NewReader(content))
			return (*XmlNode)(d), err
		}
	})
}

type XmlNode html.Node

func (n *XmlNode) Find(expr string) ([]*XmlNode, error) {
	hns, err := htmlquery.QueryAll((*html.Node)(n), strings.ToLower(expr)) // 这里的 xpath 表达式需要转换为小写，否则可能匹配不到
	if err != nil {
		return nil, err
	}

	var xns []*XmlNode
	for _, d := range hns {
		xns = append(xns, (*XmlNode)(d))
	}
	return xns, nil
}

func (n *XmlNode) FindOne(expr string) (*XmlNode, error) {
	d, err := htmlquery.Query((*html.Node)(n), strings.ToLower(expr))
	return (*XmlNode)(d), err
}

func (n *XmlNode) InnerText() string {
	return htmlquery.InnerText((*html.Node)(n))
}

func (n *XmlNode) ToString() string {
	return htmlquery.OutputHTML((*html.Node)(n), true)
}
