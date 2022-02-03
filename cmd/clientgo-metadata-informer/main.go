package main

import (
	"context"
	"flag"
	"strconv"
	"time"

	gologr "github.com/go-logr/logr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	logr      gologr.Logger
	namespace string
)

func main() {
	ctx := gologr.NewContext(context.TODO(), logr)

	c := metadata.NewForConfigOrDie(ctrl.GetConfigOrDie())
	informerFactory := metadatainformer.NewFilteredSharedInformerFactory(c, time.Hour, namespace, nil)
	i := informerFactory.ForResource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	})
	i.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			name, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				logr.Error(err, "while extracting object name")
				return
			}
			logr.Info("add", "name", name)
		},
	})
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	time.Sleep(time.Hour)
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
