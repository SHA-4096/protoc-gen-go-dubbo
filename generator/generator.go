/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package generator

import (
	"fmt"
	"strings"
)

import (
	"google.golang.org/protobuf/compiler/protogen"
)

import (
	"github.com/dubbogo/protoc-gen-go-dubbo/util"
)

func GenDubbo(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	genPreamble(g, dubboGo)
	genPackage(g, dubboGo)
	genImports(g, dubboGo)
	genConst(g, dubboGo)
	genTypeCheck(g, dubboGo)
	genInterface(g, dubboGo)
	genInterfaceImpl(g, dubboGo)
	genMethodInfo(g, dubboGo)
	genHandler(g, dubboGo)
	genServiceInfo(g, dubboGo)
}

func genPreamble(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	g.P("// Code generated by protoc-gen-go-dubbo. DO NOT EDIT.")
	g.P()
	g.P("// Source: ", dubboGo.Source)
	g.P("// Package: ", strings.ReplaceAll(dubboGo.ProtoPackage, ".", "_"))
	g.P()
}

func genPackage(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	g.P("package ", dubboGo.GoPackageName)
	g.P()
}

func genImports(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	g.P(`
		import (
			"context"

			"dubbo.apache.org/dubbo-go/v3"
			"dubbo.apache.org/dubbo-go/v3/client"
			"dubbo.apache.org/dubbo-go/v3/common"
			"dubbo.apache.org/dubbo-go/v3/common/constant"
			"dubbo.apache.org/dubbo-go/v3/server"
		)
	`)
}

func genConst(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	g.P("const (")
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf("// %sName is the fully-qualified name of the %s service.", s.ServiceName, s.ServiceName))
		g.P(fmt.Sprintf(`%sName = "%s"`, s.ServiceName, s.InterfaceName))
		g.P()

		g.P(`
// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.`)
		for _, m := range s.Methods {
			g.P(fmt.Sprintf(`// %s%sProcedure is the fully-qualified name of the %s's %s RPC.'`,
				s.ServiceName, m.MethodName, s.ServiceName, m.MethodName))
			g.P(fmt.Sprintf(`%s%sProcedure = "/%s/%s"`, s.ServiceName, m.MethodName, s.InterfaceName, m.InvokeName))
		}
	}

	g.P(")")
	g.P()
}

func genTypeCheck(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	g.P("var (")
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf(`_ %s = (*%sImpl)(nil)`, s.ServiceName, s.ServiceName))
	}
	g.P(")")
	g.P()
}

func genInterface(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf("type %s interface {", s.ServiceName))
		for _, m := range s.Methods {
			g.P(fmt.Sprintf("%s(ctx context.Context, %s opts ...client.CallOption) (%s, error)",
				util.ToUpper(m.MethodName), buildRequestArgs(m, true), buildReturnType(m)))
		}
		g.P("}")
		g.P()
	}
}

func genInterfaceImpl(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf("// New%s constructs a client for the %s service", s.ServiceName, s.InterfaceName))
		g.P(fmt.Sprintf("func New%s(cli *client.Client, opts ...client.ReferenceOption) (%s, error) {",
			s.ServiceName, s.ServiceName))
		g.P(fmt.Sprintf(`conn, err := cli.DialWithInfo("%s", &%s_ClientInfo, opts...)`,
			s.InterfaceName, s.ServiceName))
		g.P("if err != nil {")
		g.P("return nil, err")
		g.P("}")
		g.P(fmt.Sprintf("return &%sImpl {", s.ServiceName))
		g.P("conn: conn,")
		g.P("}, nil")
		g.P("}")
		g.P()

		g.P("func SetConsumerService(srv common.RPCService) {")
		g.P(fmt.Sprintf("dubbo.SetConsumerServiceWithInfo(srv, &%s_ClientInfo)", s.ServiceName))
		g.P("}")
		g.P()

		g.P(fmt.Sprintf("// %sImpl implements %s", s.ServiceName, s.ServiceName))
		g.P(fmt.Sprintf("type %sImpl struct {", s.ServiceName))
		g.P("conn *client.Connection")
		g.P("}")
		g.P()

		for _, m := range s.Methods {
			g.P(fmt.Sprintf("func (c *%sImpl) %s(ctx context.Context, %s opts ...client.CallOption) (%s, error) {",
				s.ServiceName, util.ToUpper(m.MethodName), buildRequestArgs(m, true), buildReturnType(m)))
			g.P(fmt.Sprintf("resp := new(%s)", m.ReturnType))
			g.P(fmt.Sprintf(`if err := c.conn.CallUnary(ctx, []interface{}{%s}, resp, "%s", opts...); err != nil {`,
				buildRequestInvokeArgs(m), m.InvokeName))
			g.P(fmt.Sprintf("return %s, err", util.DefaultValue(m.ReturnType)))
			g.P("}")
			if util.IsBasicType(m.ReturnType) {
				g.P("return *resp, nil")
			} else {
				g.P("return resp, nil")
			}
			g.P("}")
			g.P()
		}
	}
}

func genMethodInfo(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf("var %s_ClientInfo = client.ClientInfo {", s.ServiceName))
		g.P(fmt.Sprintf(`InterfaceName: "%s",`, s.InterfaceName))
		g.P("MethodNames: []string {")
		for _, m := range s.Methods {
			g.P(fmt.Sprintf(`"%s",`, m.InvokeName))
		}
		g.P("},")
		g.P("ConnectionInjectFunc: func(dubboCliRaw interface{}, conn *client.Connection) {")
		g.P(fmt.Sprintf("dubboCli := dubboCliRaw.(*%sImpl)", s.ServiceName))
		g.P("dubboCli.conn = conn")
		g.P("},")
		g.P("}")
		g.P()
	}
}

func genHandler(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf("// %sHandler is an implementation of the %s service.", s.ServiceName, s.InterfaceName))
		g.P(fmt.Sprintf("type %sHandler interface {", s.ServiceName))
		for _, m := range s.Methods {
			g.P(fmt.Sprintf("%s(ctx context.Context, %s) (%s, error)", util.ToUpper(m.MethodName), buildRequestArgs(m, false), buildReturnType(m)))
		}
		g.P("}")
		g.P()

		g.P(fmt.Sprintf("func Register%sHandler(srv *server.Server, hdlr %sHandler, opts ...server.ServiceOption) error {", s.ServiceName, s.ServiceName))
		g.P(fmt.Sprintf("return srv.Register(hdlr, &%s_ServiceInfo, opts...)", s.ServiceName))
		g.P("}")
		g.P()

		g.P("func SetProviderService(srv common.RPCService) {")
		g.P(fmt.Sprintf("dubbo.SetProviderServiceWithInfo(srv, &%s_ServiceInfo)", s.ServiceName))
		g.P("}")
		g.P()
	}
}

func genServiceInfo(g *protogen.GeneratedFile, dubboGo *Dubbogo) {
	for _, s := range dubboGo.Services {
		g.P(fmt.Sprintf("var %s_ServiceInfo = server.ServiceInfo {", s.ServiceName))
		g.P(fmt.Sprintf(`InterfaceName: "%s",`, s.InterfaceName))
		g.P(fmt.Sprintf("ServiceType: (*%sHandler)(nil),", s.ServiceName))
		g.P("Methods: []server.MethodInfo {")
		for _, m := range s.Methods {
			g.P("{")
			g.P(fmt.Sprintf(`Name: "%s",`, m.InvokeName))
			g.P("Type: constant.CallUnary,")
			g.P("ReqInitFunc: func() interface{} {")
			g.P(fmt.Sprintf("return new(%s)", m.ReturnType))
			g.P("},")
			g.P("MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {")
			if m.RequestExtendArgs {
				for i := range m.ArgsType {
					g.P(fmt.Sprintf("%s := args[%d].(%s)", m.ArgsName[i], i, m.ArgsType[i]))
				}
			} else {
				g.P(fmt.Sprintf("req := args[0].(*%s)", m.RequestType))
			}
			g.P(fmt.Sprintf("res, err := handler.(%sHandler).%s(ctx, %s)", s.ServiceName, util.ToUpper(m.MethodName), buildRequestInvokeArgs(m)))
			g.P("return res, err")
			g.P("},")
			g.P("},")
		}
		g.P("},")
		g.P("}")
	}
}

func buildRequestInvokeArgs(m *Method) string {
	if m.RequestExtendArgs {
		var res string
		for i, name := range m.ArgsName {
			if i == len(m.ArgsName)-1 {
				res += name
			} else {
				res += name + ", "
			}
		}
		return res
	}
	return "req"
}

func buildRequestArgs(m *Method, trailingComma bool) string {
	if m.RequestExtendArgs {
		var res string
		for i := range m.ArgsType {
			res += fmt.Sprintf("%s %s", m.ArgsName[i], m.ArgsType[i])
			if i == len(m.ArgsType)-1 && !trailingComma {
				break
			}
			res += ", "
		}
		return res
	}
	return fmt.Sprintf("req *%s,", m.RequestType)
}

func buildReturnType(m *Method) string {
	if m.ResponseExtendArgs {
		return m.ReturnType
	}
	return "*" + m.ReturnType
}
