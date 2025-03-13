package informers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/hestia-operator/internal"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DynamicResourceEvent struct {
	Type      watch.EventType
	Object    *unstructured.Unstructured
	ObjectOld *unstructured.Unstructured
	GVR       schema.GroupVersionResource
}

// DynamicInformer manages a dynamic informer
type DynamicInformer struct {
	Events   chan event.TypedGenericEvent[DynamicResourceEvent]
	client   *dynamic.DynamicClient
	gvr      schema.GroupVersionResource
	stopChan chan struct{}
	mu       sync.Mutex
	logger   logr.Logger
}

func NewDynamicInformer(client *dynamic.DynamicClient, gvr schema.GroupVersionResource) *DynamicInformer {
	return &DynamicInformer{
		client:   client,
		gvr:      gvr,
		stopChan: make(chan struct{}),
		logger:   log.Log.WithName(fmt.Sprintf("[%s]", gvr.Resource)),
	}
}

func (c *DynamicInformer) factory() dynamicinformer.DynamicSharedInformerFactory {
	return dynamicinformer.NewFilteredDynamicSharedInformerFactory(c.client, 30*time.Second, "", func(options *v1.ListOptions) {
		options.LabelSelector = fmt.Sprintf("%s=%v", internal.RunnerLabel, true)
	})
}

// Start runs the dynamic informer
func (c *DynamicInformer) Start(ctx context.Context) error {
	c.logger.Info("starting...")
	c.mu.Lock()
	c.stopChan = make(chan struct{})
	c.mu.Unlock()

	informer := c.factory().ForResource(c.gvr).Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.logger.Info("Added: " + obj.(client.Object).GetName())
			c.Events <- event.TypedGenericEvent[DynamicResourceEvent]{
				Object: DynamicResourceEvent{
					Type:   watch.Added,
					Object: obj.(*unstructured.Unstructured),
					GVR:    schema.GroupVersionResource{},
				},
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.logger.Info("Updated: " + newObj.(client.Object).GetName())
			c.Events <- event.TypedGenericEvent[DynamicResourceEvent]{
				Object: DynamicResourceEvent{
					Type:      watch.Modified,
					Object:    newObj.(*unstructured.Unstructured),
					ObjectOld: oldObj.(*unstructured.Unstructured),
					GVR:       schema.GroupVersionResource{},
				},
			}
		},
		DeleteFunc: func(obj interface{}) {
			c.logger.Info("Deleted: " + obj.(client.Object).GetName())
			c.Events <- event.TypedGenericEvent[DynamicResourceEvent]{
				Object: DynamicResourceEvent{
					Type:   watch.Deleted,
					Object: obj.(*unstructured.Unstructured),
					GVR:    schema.GroupVersionResource{},
				},
			}
		},
	})

	if err != nil {
		c.logger.Error(err, "unable to add informer to informer")
		return err
	}

	// Run the informer in a separate goroutine
	go func() {
		informer.Run(c.stopChan) // Blocking call
		if ctx.Err() == nil {
			c.logger.Info("stopped unexpectedly")
			_ = c.Start(ctx)
		}
	}()

	// Wait until the manager stops
	<-ctx.Done()
	c.logger.Info("manager closed, shutting down...")
	c.Stop()
	return nil
}

func (c *DynamicInformer) Stop() {
	c.mu.Lock()
	if c.stopChan != nil {
		defer c.mu.Unlock()
		close(c.stopChan)
		c.stopChan = nil
	}
	c.logger.Info("stopped!")
}
