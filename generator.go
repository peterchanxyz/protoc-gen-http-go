package main

import (
	"os"
	"path/filepath"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	bytesPackage   = protogen.GoImportPath("bytes")
	contextPackage = protogen.GoImportPath("context")
	errorsPkg      = protogen.GoImportPath("errors")
	jsonPackage    = protogen.GoImportPath("encoding/json")
	fmtPackage     = protogen.GoImportPath("fmt")
	ioPackage      = protogen.GoImportPath("io")
	mimePackage    = protogen.GoImportPath("mime")
	httpPackage    = protogen.GoImportPath("net/http")
	strconvPackage = protogen.GoImportPath("strconv")
	stringsPackage = protogen.GoImportPath("strings")
	reflectPackage = protogen.GoImportPath("reflect")
)

// generateFile generates a _gin.pb.go file.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	filename := file.GeneratedFilenamePrefix + "_http.pb.go"
	programName := filepath.Base(os.Args[0])
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by ", programName, ". DO NOT EDIT.")
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()

	for _, service := range file.Services {
		genService(g, service)
	}
	return g
}

func genService(g *protogen.GeneratedFile, s *protogen.Service) {
	if isDeprecatedService(s) {
		g.P("//")
		deprecated(g)
	}

	// service server interface
	g.P("// ", s.GoName, "Server is the server API for ", s.GoName, " service.")
	g.P("type ", s.GoName, "Server interface {")

	for _, method := range s.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue
		}
		g.P("    ", method.GoName, "(", contextPackage.Ident("Context"), ", *", method.Input.GoIdent, ") (*", method.Output.GoIdent, ", error)")
	}
	g.P("}")
	g.P()

	genWriteErr(g)
	g.P()
	genWriteRsp(g)

	// g.P("// ", s.GoName, "RegisterHttpServer has a ", s.GoName, "HTTPService interface to http.HandlerFunc.")
	g.P("func RegisterHttpServer(srv any, impl ", s.GoName, "Server) (err error) {")
	g.P("    mux, ok := srv.(interface { Handle(string, ", httpPackage.Ident("Handler"), ") })")
	g.P("    if !ok {")
	g.P("        err = ", errorsPkg.Ident("New"), "(\"srv must implement HttpServerMux\")")
	g.P("        return")
	g.P("    }")

	for _, method := range s.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue
		}
		g.P("    mux.Handle(", method.GoName, "Handler(impl))")
	}
	g.P("    return")
	g.P("}")
	g.P()

	for _, method := range s.Methods {
		if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
			continue
		}
		genMethod(g, method)
	}
}

func genMethod(g *protogen.GeneratedFile, m *protogen.Method) (err error) {
	g.P("// ", m.GoName, " returns ", m.Parent.GoName, "HTTPService interface's ", m.GoName, " converted to http.HandlerFunc.")
	if m.Comments.Leading.String() != "" {
		g.P("//")
	}
	var pattern string
	var pathParams []*pathParam
	rule, ok := proto.GetExtension(m.Desc.Options(), annotations.E_Http).(*annotations.HttpRule)
	if ok {
		httpMtd, path := buildHTTPRule(m, rule)
		pattern = httpMtd + " " + path
		pathParams, err = parsePathParam(path)
		if err != nil {
			return err
		}
	}

	g.P("func ", m.GoName, "Handler(srv ", m.Parent.GoName, "Server) (pattern string, hdr ", httpPackage.Ident("Handler"), ") {")
	g.P("	pattern = ", "\"", pattern, "\"")
	g.P("	hdr = ", httpPackage.Ident("HandlerFunc"), "(func(w ", httpPackage.Ident("ResponseWriter"), ", r *", httpPackage.Ident("Request"), ") {")
	g.P("		ctx := r.Context()")
	g.P()
	g.P("		in := &", m.Input.GoIdent, "{}")
	g.P("		if r.Method != ", httpPackage.Ident("MethodGet"), " {")
	g.P("			reqba, err := ", ioPackage.Ident("ReadAll"), "(r.Body)")
	g.P("			if err != nil {")
	g.P("				writeErr(w, err)")
	g.P("				return")
	g.P("			}")
	g.P("			err = ", jsonPackage.Ident("Unmarshal"), "(reqba, in)")
	g.P("			if err != nil {")
	g.P("				writeErr(w, err)")
	g.P("				return")
	g.P("			}")
	g.P("		}")

	for _, t := range pathParams {
		g.P("in.", t.GoName, " = r.PathValue(\"", t.Name, "\")")
	}

	g.P("		out, err := srv.", m.GoName, "(ctx, in)")
	g.P("		if err != nil {")
	g.P("			writeErr(w, err)")
	g.P("			return")
	g.P("		}")
	g.P("		writeRsp(w, out)")
	g.P("    })")
	g.P("    return")
	g.P("}")
	g.P()

	return nil
}

func genWriteErr(g *protogen.GeneratedFile) {
	g.P("func writeErr(w ", httpPackage.Ident("ResponseWriter"), ", err error) {")
	g.P("    w.Header().Set(\"Content-Type\", \"application/json\")")
	g.P("	 w.WriteHeader(", httpPackage.Ident("StatusBadRequest"), ")")
	g.P("    errRst := map[string]any{}")
	g.P("    errRst[\"msg\"] = err.Error()")
	g.P("    if cerr, ok := err.(interface{ Code() int }); ok {")
	g.P("        errRst[\"code\"] = cerr.Code()")
	g.P("    }")
	g.P("	respba, err := ", jsonPackage.Ident("Marshal"), "(resp)")
	g.P("	w.Write([]byte(`{\"error\":` + err.Error() + `}`))")
	g.P("}")
}

func genWriteRsp(g *protogen.GeneratedFile) {
	g.P("func writeRsp(w ", httpPackage.Ident("ResponseWriter"), ", resp any) {")
	g.P("	 w.Header().Set(\"Content-Type\", \"application/json\")")
	g.P("	 w.WriteHeader(", httpPackage.Ident("StatusOK"), ")")
	g.P("	 respba, err := ", jsonPackage.Ident("Marshal"), "(resp)")
	g.P("	 if err != nil {")
	g.P("		w.WriteErr(e, err)")
	g.P("		return")
	g.P("    }")
	g.P("	 w.Write(respba)")
	g.P("}")
}

func isDeprecatedService(service *protogen.Service) bool {
	serviceOptions, ok := service.Desc.Options().(*descriptorpb.ServiceOptions)
	return ok && serviceOptions.GetDeprecated()
}

func isDeprecatedMethod(method *protogen.Method) bool {
	methodOptions, ok := method.Desc.Options().(*descriptorpb.MethodOptions)
	return ok && methodOptions.GetDeprecated()
}

func deprecated(g *protogen.GeneratedFile) {
	g.P("// Deprecated: do not use.")
}

func buildHTTPRule(m *protogen.Method, rule *annotations.HttpRule) (method string, path string) {
	switch pattern := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		path = pattern.Get
		method = "GET"
	case *annotations.HttpRule_Put:
		path = pattern.Put
		method = "PUT"
	case *annotations.HttpRule_Post:
		path = pattern.Post
		method = "POST"
	case *annotations.HttpRule_Delete:
		path = pattern.Delete
		method = "DELETE"
	case *annotations.HttpRule_Patch:
		path = pattern.Patch
		method = "PATCH"
	case *annotations.HttpRule_Custom:
		path = pattern.Custom.Path
		method = pattern.Custom.Kind
	}
	return
}
