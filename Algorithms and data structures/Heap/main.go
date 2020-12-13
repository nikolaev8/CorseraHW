package main

import (
	"fmt"
)

func main() {
	var n int
	_, err := fmt.Scan(&n)
	if err != nil {
		panic(err)
	}

	arr := make([]float64, n)
	for i := range arr {
		_, err := fmt.Scan(&arr[i])
		if err != nil {
			panic(err)
		}
	}

	_, cache := BuildHeap(arr)

	fmt.Println(len(cache))
	for _, swapPair := range cache {
		fmt.Println(swapPair[0], swapPair[1])
	}
}
