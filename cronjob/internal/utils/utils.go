package utils

import (
	"strconv"
	"strings"

	"github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"
	networkingv1 "k8s.io/api/networking/v1"
)

var logger = loggerpkg.GetNamedLogger("ingresswatcher")

// common way to do create unique key in Kubernetes - use the namespace and the name of the resource, joined by a delimiter. it's like cache.MetaNamespaceKeyFunc(obj).
func IngressKey(ing *networkingv1.Ingress) string {
	logger.Debug("ingressKey")

	return ing.Namespace + "/" + ing.Name
}

// ChangeSecretName check if the name has "-vX" suffix for example (-v1), if not - add it. if it have - change it to "-vX+1".
func ChangeSecretName(secretName string) (string, error) {
	logger.Debug("checkAndChangeSecretName")

	if !isStrHasVxSuffix(secretName) {
		// add "-vX" suffix to secret name
		return AddVxSuffixToStr(secretName), nil
	} else {
		// change "-vX" suffix to "-vX+1" suffix
		newSecretName, err := IncVxSuffixToStr(secretName)
		if err != nil {
			logger.Errorf("Failed to increment secret name suffix: %v", err)
			return "", err
		}
		return newSecretName, nil
	}
}

// check if the string contain the substring "-vX" in it last part, for example: "my-secret-v1"
func isStrHasVxSuffix(str string) bool {
	logger.Debug("isStrHasVxSuffix")

	// split the string to parts by "-"
	strParts := strings.Split(str, "-")

	// check if the last part contain the substring "-vX"
	if strings.Contains(strParts[len(strParts)-1], "-v") {
		return true
	}

	return false
}

// add "-vX" suffix to string name
func AddVxSuffixToStr(str string) string {
	logger.Debug("AddVxSuffixToStr")

	return str + "-v1"
}

// change "-vX" suffix to "-vX+1" suffix
func IncVxSuffixToStr(str string) (string, error) {
	logger.Debug("IncVxSuffixToStr")

	// split the string to parts by "-"
	strParts := strings.Split(str, "-")

	// split the last part to parts by "v"
	strPartsV := strings.TrimPrefix(strParts[len(strParts)-1], "v")

	// convert the last part to int
	strPartsVInt, err := strconv.Atoi(strPartsV)
	if err != nil {
		logger.Errorf("Failed to convert string to int: %v", err)
		return "", err
	}

	// change the last part to "-v(X+1)"
	strParts[len(strParts)-1] = "v" + strconv.Itoa(strPartsVInt+1)

	// join the string parts back to one string
	return strings.Join(strParts, "-"), nil
}
