package connectinject

import (
	"fmt"
	"strconv"

	"github.com/miekg/dns"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func (w *MeshWebhook) configureDNS(pod *corev1.Pod) error {
	// First, we need to determine the nameservers configured in this cluster from /etc/resolv.conf.
	etcResolvConf := "/etc/resolv.conf"
	if w.etcResolvFile != "" {
		etcResolvConf = w.etcResolvFile
	}
	cfg, err := dns.ClientConfigFromFile(etcResolvConf)
	if err != nil {
		return err
	}

	// Set DNS policy on the pod to None because we want DNS to work according to the config we will provide.
	pod.Spec.DNSPolicy = corev1.DNSNone

	// Set the consul-dataplane's DNS server as the first server in the list (i.e. localhost).
	// We want to do that so that when consul cannot resolve the record, we will fall back to the nameservers
	// configured in our /etc/resolv.conf. It's important to add Consul DNS as the first nameserver because
	// if we put kube DNS first, it will return NXDOMAIN response and a DNS client will not fall back to other nameservers.
	if pod.Spec.DNSConfig == nil {
		consulDPAddress := "127.0.0.1"
		nameservers := []string{consulDPAddress}
		nameservers = append(nameservers, cfg.Servers...)
		var options []corev1.PodDNSConfigOption
		if cfg.Ndots != 0 {
			ndots := strconv.Itoa(cfg.Ndots)
			options = append(options, corev1.PodDNSConfigOption{
				Name:  "ndots",
				Value: &ndots,
			})
		}
		if cfg.Timeout != 0 {
			options = append(options, corev1.PodDNSConfigOption{
				Name:  "timeout",
				Value: pointer.String(strconv.Itoa(cfg.Timeout)),
			})
		}
		if cfg.Attempts != 0 {
			options = append(options, corev1.PodDNSConfigOption{
				Name:  "attempts",
				Value: pointer.String(strconv.Itoa(cfg.Attempts)),
			})
		}

		pod.Spec.DNSConfig = &corev1.PodDNSConfig{
			Nameservers: nameservers,
			Searches:    cfg.Search,
			Options:     options,
		}
	} else {
		return fmt.Errorf("DNS redirection to Consul is not supported with an already defined DNSConfig on the pod")
	}
	return nil
}
