package kubernetes

import (
	"context"
	"sync"
	"time"

	"github.com/ebay/collectbeat/discoverer"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/api/v1"
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
	podRunners          podRunners
	builders            *discoverer.Builders
}

type podRunners struct {
	sync.Mutex
	pods map[string]*corev1.Pod
}

type NodeOption struct{}

// NewPodWatcher initializes the watcher factory to provide a local state of
// runners from the cluster (filtered to the given host)
func NewPodWatcher(kubeClient *k8s.Client, builders *discoverer.Builders, syncPeriod time.Duration, host string) *PodWatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &PodWatcher{
		kubeClient:          kubeClient,
		builders:            builders,
		syncPeriod:          syncPeriod,
		podQueue:            make(chan *corev1.Pod, 10),
		nodeFilter:          k8s.QueryParam("fieldSelector", "spec.nodeName="+host),
		lastResourceVersion: "0",
		ctx:                 ctx,
		stop:                cancel,
		podRunners: podRunners{
			pods: make(map[string]*corev1.Pod),
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

func (p *PodWatcher) onPodAdd(pod *corev1.Pod) {
	p.podRunners.Lock()
	defer p.podRunners.Unlock()

	p.podRunners.pods[pod.Metadata.GetUid()] = pod
	p.builders.StartModuleRunners(pod)

}

func (p *PodWatcher) onPodUpdate(pod *corev1.Pod) {
	oldPod := p.GetPod(pod.Metadata.GetUid())
	if oldPod.Metadata.GetResourceVersion() != pod.Metadata.GetResourceVersion() {
		// Process the new pod changes
		p.onPodDelete(oldPod)
		p.onPodAdd(pod)
	}
}

func (p *PodWatcher) onPodDelete(pod *corev1.Pod) {
	p.podRunners.Lock()
	defer p.podRunners.Unlock()

	// This makes sure that we have an IP in hand in case the notification came in late
	oldPo, ok := p.podRunners.pods[pod.Metadata.GetUid()]
	if ok {
		p.builders.StopModuleRunners(oldPo)
		delete(p.podRunners.pods, pod.Metadata.GetUid())
	}

}

func (p *PodWatcher) worker() {
	for pod := range p.podQueue {
		if pod.Metadata.GetDeletionTimestamp() != nil {
			p.onPodDelete(pod)
		} else {
			existing := p.GetPod(pod.Metadata.GetUid())
			if existing != nil {
				p.onPodUpdate(pod)
			} else {
				p.onPodAdd(pod)
			}
		}
	}

}

func (p *PodWatcher) GetPod(uid string) *corev1.Pod {
	p.podRunners.Lock()
	defer p.podRunners.Unlock()
	return p.podRunners.pods[uid]
}

func (p *PodWatcher) Stop() {
	p.stop()
	close(p.podQueue)
}
