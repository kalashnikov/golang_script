package main

import (
	"fmt"
	"math"
	"sync"
)

const limit = 10000

// Send the sequence 2, 3, 4, ... to channel 'ch'.
func Generate(ch chan<- int) {
	for i := 2; ; i++ {
		ch <- i // Send 'i' to channel 'ch'.
	}
}

// Copy the values from channel 'in' to channel 'out',
// removing those divisible by 'prime'.
func Filter(in <-chan int, out chan<- int, prime int) {
	for {
		i := <-in // Receive value from 'in'.
		if i%prime != 0 {
			out <- i // Send 'i' to 'out'.
		}
	}
}

// The prime sieve: Daisy-chain Filter processes.
func Sieve(limit int, result map[int]bool) {
	ch := make(chan int) // Create a new channel.
	go Generate(ch)      // Launch Generate goroutine.
	for {
		prime := <-ch
		if prime > limit {
			break
		}
		result[prime] = false
		ch1 := make(chan int)
		go Filter(ch, ch1, prime)
		ch = ch1
	}
}

func PrimeFactor(t int, primes map[int]bool, result map[float64]float64) {
	for prime := range primes {
		for t%prime == 0 {
			t = t / prime
			result[float64(prime)]++
		}

		if t <= 1 {
			return
		}
	}
}

func EulerTotient(t int, primes map[int]bool, data *float64, wg *sync.WaitGroup, mutex *sync.Mutex) {
	defer wg.Done()

	result := make(map[float64]float64)
	PrimeFactor(t, primes, result)

	ans := 1.0
	for k, v := range result {
		ans *= (k - 1.0) * math.Pow(k, (v-1.0))
	}

	mutex.Lock()
	*data += ans
	mutex.Unlock()
}

func Euler72() {
	limit := 1000000
	// =============================================
	// Euler 72
	//
	// Brute-Force to Find EulerTotient, too slow
	//primes := make(map[int]bool)
	//Sieve(limit, primes)
	//ans := 0.0
	//var wg sync.WaitGroup
	//var mutex = &sync.Mutex{}
	//for i := 2; i <= limit; i++ {
	//	wg.Add(1)
	//	EulerTotient(i, primes, &ans, &wg, mutex)
	//}
	//wg.Wait()
	//fmt.Println(ans)

	// https://projecteuler.net/overview=072
	phis := make([]float64, limit+1)
	for n := 2; n <= limit; n++ {
		phis[n] = float64(n)
	}
	for n := 2; n <= limit; n++ {
		if phis[n] == float64(n) {
			for m := n; m <= limit; m += n {
				phis[m] = phis[m] - phis[m]/float64(n)
			}
		}
	}
	ans2 := 0.0
	for _, v := range phis {
		ans2 += v
	}
	fmt.Println(ans2)
	// =============================================
}

func main() {
	Euler72()
}
