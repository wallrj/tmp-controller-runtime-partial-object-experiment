package main

import (
	"context"
	"flag"
	"strconv"

	gologr "github.com/go-logr/logr"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	logr      gologr.Logger
	namespace string
)

func main() {
	ctx := gologr.NewContext(context.TODO(), logr)

	c, err := cache.New(ctrl.GetConfigOrDie(), cache.Options{
		Namespace: namespace,
	})
	if err != nil {
		panic(err)
	}
	o := &v1.PartialObjectMetadata{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	})
	i, err := c.GetInformer(ctx, o)
	if err != nil {
		panic(err)
	}
	i.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			name, err := toolscache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				logr.Error(err, "while extracting object name")
				return
			}
			logr.Info("add", "name", name)
		},
	})

	if err := c.Start(ctx); err != nil {
		panic(err)
	}
}

func init() {
	flag.StringVar(&namespace, "namespace", "", "watch one namespace")

	opts := ctrlzap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	logr = ctrlzap.New(ctrlzap.UseFlagOptions(&opts))

	atomlvl, ok := opts.Level.(zap.AtomicLevel)
	if ok {
		zaplvl := atomlvl.Level()
		kloglvl := 0
		if zaplvl < 0 {
			kloglvl = -int(zaplvl)
		}
		dummy := flag.FlagSet{}
		klog.InitFlags(&dummy)

		// No way those can fail, so let's just ignore the errors.
		_ = dummy.Set("v", strconv.Itoa(kloglvl))
		_ = dummy.Parse(nil)
	}
	klog.SetLogger(logr)
	ctrl.SetLogger(logr)
}
