package kubernetes

import (
	"context"
	"sync"
	"time"

	"github.com/ebay/collectbeat/discoverer"
	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

// PodWatcher is a controller that synchronizes Pods.
type PodWatcher struct {
	kubeClient          *k8s.Client
	syncPeriod          time.Duration
	podQueue            chan *corev1.Pod
	nodeFilter          k8s.Option
	lastResourceVersion string
	ctx                 context.Context
	stop                context.CancelFunc
	pods                podMeta
	builders            *discoverer.Builders
	indexers            *kubernetes.Indexers
}

type podMeta struct {
	sync.RWMutex
	pods        map[string]*kubernetes.Pod
	annotations map[string]common.MapStr
}

func (p *podMeta) AddPod(name string, pod *kubernetes.Pod) {
	p.Lock()
	defer p.Unlock()

	p.pods[name] = pod
}

func (p *podMeta) GetPod(name string) (*kubernetes.Pod, bool) {
	p.RLock()
	defer p.RUnlock()

	val, ok := p.pods[name]
	return val, ok
}

func (p *podMeta) DeletePod(name string) {
	p.Lock()
	defer p.Unlock()

	delete(p.pods, name)
}

func (p *podMeta) AddPodAnnotations(name string, meta common.MapStr) {
	p.Lock()
	defer p.Unlock()

	p.annotations[name] = meta
}

func (p *podMeta) GetPodAnnotations(name string) (common.MapStr, bool) {
	p.RLock()
	defer p.RUnlock()

	val, ok := p.annotations[name]
	return val, ok
}

func (p *podMeta) DeletePodAnnotations(name string) {
	p.Lock()
	defer p.Unlock()

	delete(p.annotations, name)
}

type NodeOption struct{}

// NewPodWatcher initializes the watcher factory to provide a local state of
// runners from the cluster (filtered to the given host)
func NewPodWatcher(kubeClient *k8s.Client, indexers *kubernetes.Indexers, syncPeriod time.Duration, host string) *PodWatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &PodWatcher{
		kubeClient:          kubeClient,
		syncPeriod:          syncPeriod,
		podQueue:            make(chan *corev1.Pod, 10),
		nodeFilter:          k8s.QueryParam("fieldSelector", "spec.nodeName="+host),
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
		indexers:            indexers,
		pods: podMeta{
			pods:        make(map[string]*kubernetes.Pod),
			annotations: make(map[string]common.MapStr),
		},
	}
}

func (p *PodWatcher) syncPods() error {
	logp.Info("kubernetes: %s", "Performing a pod sync")
	pods, err := p.kubeClient.CoreV1().ListPods(
		p.ctx,
		"",
		p.nodeFilter,
		k8s.ResourceVersion(p.lastResourceVersion))

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		p.podQueue <- pod
	}

	// Store last version
	p.lastResourceVersion = pods.Metadata.GetResourceVersion()

	logp.Info("kubernetes: %s", "Pod sync done")
	return nil
}

func (p *PodWatcher) watchPods() {
	for {
		logp.Info("kubernetes: %s", "Watching API for pod events")
		watcher, err := p.kubeClient.CoreV1().WatchPods(p.ctx, "", p.nodeFilter)
		if err != nil {
			//watch pod failures should be logged and gracefully failed over as metadata retrieval
			//should never stop.
			logp.Err("kubernetes: Watching API eror %v", err)
			time.Sleep(time.Second)
			continue
		}
		for {
			_, pod, err := watcher.Next()
			if err != nil {
				logp.Err("kubernetes: Watching API eror %v", err)
				time.Sleep(time.Second)
				break
			}

			p.podQueue <- pod
		}
	}

}

func (p *PodWatcher) Run() bool {
	// Start pod processing worker:
	go p.worker()

	// Make sure that events don't flow into the annotator before informer is fully set up
	// Sync initial state:
	synced := make(chan struct{})
	go func() {
		p.syncPods()
		close(synced)
	}()

	select {
	case <-time.After(ready_timeout):
		p.Stop()
		return false
	case <-synced:
		// Watch for new changes
		go p.watchPods()
		return true
	}
}

func (p *PodWatcher) onPodAdd(pod *kubernetes.Pod) {
	for _, m := range p.indexers.GetMetadata(pod) {
		p.pods.AddPodAnnotations(m.Index, m.Data)
	}

	p.pods.AddPod(pod.Metadata.UID, pod)
	p.builders.StartModuleRunners(pod)

}

func (p *PodWatcher) onPodUpdate(pod *kubernetes.Pod) {
	oldPod := p.GetPod(pod.Metadata.UID)
	if oldPod.Metadata.ResourceVersion != pod.Metadata.ResourceVersion {
		// Process the new pod changes
		p.onPodDelete(oldPod)
		p.onPodAdd(pod)
	}
}

func (p *PodWatcher) onPodDelete(pod *kubernetes.Pod) {
	// This makes sure that we have an IP in hand in case the notification came in late
	oldPo, ok := p.pods.GetPod(pod.Metadata.UID)
	if ok {
		p.builders.StopModuleRunners(oldPo)
		p.pods.DeletePod(pod.Metadata.UID)
	}

	for _, index := range p.indexers.GetIndexes(pod) {
		p.pods.DeletePodAnnotations(index)
	}
}

func (p *PodWatcher) worker() {
	for po := range p.podQueue {
		pod := kubernetes.GetPodMeta(po)
		if pod.Metadata.DeletionTimestamp != "" {
			p.onPodDelete(pod)
		} else {
			existing := p.GetPod(pod.Metadata.UID)
			if existing != nil {
				p.onPodUpdate(pod)
			} else {
				p.onPodAdd(pod)
			}
		}
	}

}

func (p *PodWatcher) GetPod(uid string) *kubernetes.Pod {
	po, _ := p.pods.GetPod(uid)
	return po
}

func (p *PodWatcher) Stop() {
	p.stop()
	close(p.podQueue)
}

func (p *PodWatcher) GetMetaData(arg string) common.MapStr {
	meta, ok := p.pods.GetPodAnnotations(arg)

	if ok {
		return meta
	}

	return nil
}
