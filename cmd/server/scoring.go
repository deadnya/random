package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"

	"numbers/internal/ui"
)

type spec struct {
	Key   string
	Name  string
	Value string
	Score int
}

type specOdd struct {
	Probability float64
	Score       int
}

type rarityScorer struct {
	odds map[string]specOdd
}

type specRule struct {
	Key   string
	Name  string
	Value string
	Match func(number int, facts numberFacts) bool
}

type numberFacts struct {
	digits         string
	uniqueDigits   int
	digitSum       int
	allDigitsEqual bool
	palindrome     bool
}

var specRules = []specRule{
	{
		Key:   "prime",
		Name:  "Prime Time",
		Value: "Only divisible by 1 and itself",
		Match: func(number int, _ numberFacts) bool {
			return isPrime(number)
		},
	},
	{
		Key:   "all_digits_equal",
		Name:  "Six of a Kind",
		Value: "All digits identical",
		Match: func(_ int, facts numberFacts) bool {
			return facts.allDigitsEqual
		},
	},
	{
		Key:   "palindrome",
		Name:  "Mirror Number",
		Value: "Reads the same forwards and backwards",
		Match: func(_ int, facts numberFacts) bool {
			return facts.palindrome
		},
	},
	{
		Key:   "unique_digits_6",
		Name:  "Sextuplet",
		Value: "All six digits unique",
		Match: func(_ int, facts numberFacts) bool {
			return facts.uniqueDigits == 6
		},
	},
	{
		Key:   "unique_digits_5",
		Name:  "Quintuplet",
		Value: "Exactly five unique digits",
		Match: func(_ int, facts numberFacts) bool {
			return facts.uniqueDigits == 5
		},
	},
	{
		Key:   "unique_digits_4",
		Name:  "Quadruplet",
		Value: "Exactly four unique digits",
		Match: func(_ int, facts numberFacts) bool {
			return facts.uniqueDigits == 4
		},
	},
	{
		Key:   "divisible_by_9",
		Name:  "Nine Lives",
		Value: "Digit sum divisible by 9",
		Match: func(_ int, facts numberFacts) bool {
			return facts.digitSum%9 == 0
		},
	},
	{
		Key:   "divisible_by_7",
		Name:  "Lucky Sevens",
		Value: "Divisible by 7",
		Match: func(number int, _ numberFacts) bool {
			return number%7 == 0
		},
	},
	{
		Key:   "trailing_000",
		Name:  "Trailing Zeroes",
		Value: "Ends with triple zero",
		Match: func(_ int, facts numberFacts) bool {
			return strings.HasSuffix(facts.digits, "000")
		},
	},
	{
		Key:   "exact_zero",
		Name:  "The Big Zero",
		Value: "Rolled number is exactly 000000",
		Match: func(number int, _ numberFacts) bool {
			return number == 0
		},
	},
	{
		Key:   "even_number",
		Name:  "Even Steven",
		Value: "Divisible by 2",
		Match: func(number int, _ numberFacts) bool {
			return number%2 == 0
		},
	},
	{
		Key:   "odd_number",
		Name:  "Odd Todd",
		Value: "Not divisible by 2",
		Match: func(number int, _ numberFacts) bool {
			return number%2 == 1
		},
	},
	{
		Key:   "starts_with_1",
		Name:  "Leading One",
		Value: "Begins with the loneliest number",
		Match: func(_ int, facts numberFacts) bool {
			return strings.HasPrefix(facts.digits, "1")
		},
	},
	{
		Key:   "ends_with_5",
		Name:  "High Five",
		Value: "Ends with a hand",
		Match: func(_ int, facts numberFacts) bool {
			return strings.HasSuffix(facts.digits, "5")
		},
	},
	{
		Key:   "contains_7",
		Name:  "Lucky Seven",
		Value: "Contains the lucky digit",
		Match: func(_ int, facts numberFacts) bool {
			return strings.Contains(facts.digits, "7")
		},
	},
	{
		Key:   "digit_sum_under_15",
		Name:  "Digit Sum Lite",
		Value: "Sum under 15",
		Match: func(_ int, facts numberFacts) bool {
			return facts.digitSum < 15
		},
	},
	{
		Key:   "digit_sum_over_30",
		Name:  "Digit Sum Heavy",
		Value: "Sum over 30",
		Match: func(_ int, facts numberFacts) bool {
			return facts.digitSum > 30
		},
	},
	{
		Key:   "double_digit",
		Name:  "Twin Digits",
		Value: "Has consecutive identical digits",
		Match: func(_ int, facts numberFacts) bool {
			for i := 0; i < len(facts.digits)-1; i++ {
				if facts.digits[i] == facts.digits[i+1] {
					return true
				}
			}
			return false
		},
	},
	{
		Key:   "fibonacci",
		Name:  "Fibber",
		Value: "Part of the famous sequence",
		Match: func(number int, _ numberFacts) bool {
			a, b := 0, 1
			for b < number {
				a, b = b, a+b
			}
			return b == number || number == 0
		},
	},
	{
		Key:   "perfect_square",
		Name:  "Square Root",
		Value: "Product of an integer with itself",
		Match: func(number int, _ numberFacts) bool {
			root := int(math.Sqrt(float64(number)))
			return root*root == number
		},
	},
	{
		Key:   "armstrong",
		Name:  "Armstrong's Strong",
		Value: "Sum of powered digits equals the number",
		Match: func(number int, facts numberFacts) bool {
			sum := 0
			digits := facts.digits
			n := len(digits)
			for _, ch := range digits {
				digit := int(ch - '0')
				power := 1
				for i := 0; i < n; i++ {
					power *= digit
				}
				sum += power
			}
			return sum == number
		},
	},
	{
		Key:   "ascending_digits",
		Name:  "Climbing Digits",
		Value: "Each digit higher than the last",
		Match: func(_ int, facts numberFacts) bool {
			for i := 0; i < len(facts.digits)-1; i++ {
				if facts.digits[i] >= facts.digits[i+1] {
					return false
				}
			}
			return true
		},
	},
	{
		Key:   "descending_digits",
		Name:  "Descending Digits",
		Value: "Each digit lower than the last",
		Match: func(_ int, facts numberFacts) bool {
			for i := 0; i < len(facts.digits)-1; i++ {
				if facts.digits[i] <= facts.digits[i+1] {
					return false
				}
			}
			return true
		},
	},
	{
		Key:   "contains_123",
		Name:  "Counting Up",
		Value: "Has the sequence 1-2-3",
		Match: func(_ int, facts numberFacts) bool {
			return strings.Contains(facts.digits, "123")
		},
	},
	{
		Key:   "contains_456",
		Name:  "Mid-Count",
		Value: "Has the sequence 4-5-6",
		Match: func(_ int, facts numberFacts) bool {
			return strings.Contains(facts.digits, "456")
		},
	},
	{
		Key:   "digit_product_24",
		Name:  "Dozen Doubled",
		Value: "Product of digits equals 24",
		Match: func(_ int, facts numberFacts) bool {
			product := 1
			for _, ch := range facts.digits {
				digit := int(ch - '0')
				product *= digit
			}
			return product == 24
		},
	},
	{
		Key:   "alternating_parity",
		Name:  "Odd-Even Shift",
		Value: "Digits alternate between odd and even",
		Match: func(_ int, facts numberFacts) bool {
			if len(facts.digits) < 2 {
				return false
			}
			prevEven := (int(facts.digits[0]-'0') % 2) == 0
			for i := 1; i < len(facts.digits); i++ {
				currEven := (int(facts.digits[i]-'0') % 2) == 0
				if currEven == prevEven {
					return false
				}
				prevEven = currEven
			}
			return true
		},
	},
	{
		Key:   "three_consecutive",
		Name:  "Three Consecutive",
		Value: "yes",
		Match: func(_ int, facts numberFacts) bool {
			for i := 0; i < len(facts.digits)-2; i++ {
				d1 := int(facts.digits[i] - '0')
				d2 := int(facts.digits[i+1] - '0')
				d3 := int(facts.digits[i+2] - '0')
				if d2 == d1+1 && d3 == d2+1 {
					return true
				}
			}
			return false
		},
	},
}

func secureIntn(max int64) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func (s *rarityScorer) calculate(number int) ([]spec, int) {
	facts := inspectNumber(number)
	items := make([]spec, 0, len(specRules))
	total := 0

	for _, rule := range specRules {
		s.addIfHit(&items, &total, rule.Key, rule.Name, rule.Value, rule.Match(number, facts))
	}

	return items, total
}

func (s *rarityScorer) addIfHit(items *[]spec, total *int, key, name, value string, hit bool) {
	if !hit || s == nil {
		return
	}

	odds, ok := s.odds[key]
	if !ok || odds.Score <= 0 {
		return
	}

	*items = append(*items, spec{Key: key, Name: name, Value: value, Score: odds.Score})
	*total += odds.Score
}

func inspectNumber(number int) numberFacts {
	digits := fmt.Sprintf("%06d", number)
	unique := make(map[rune]struct{}, len(digits))
	digitSum := 0

	for _, ch := range digits {
		unique[ch] = struct{}{}
		digitSum += int(ch - '0')
	}

	return numberFacts{
		digits:         digits,
		uniqueDigits:   len(unique),
		digitSum:       digitSum,
		allDigitsEqual: strings.Count(digits, string(digits[0])) == len(digits),
		palindrome:     digits == reverseString(digits),
	}
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}

	limit := int(math.Sqrt(float64(n)))
	for i := 3; i <= limit; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func reverseString(input string) string {
	runes := []rune(input)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func isRareSpec(key string) bool {
	rareSpecs := map[string]bool{
		"fibonacci": true,
		"perfect_square": true,
		"armstrong": true,
		"ascending_digits": true,
		"descending_digits": true,
		"contains_123": true,
		"contains_456": true,
		"digit_product_24": true,
		"exact_zero": true,
		"all_digits_equal": true,
		"trailing_000": true,
		"palindrome": true,
		"three_consecutive": true,
	}
	return rareSpecs[key]
}

func toUISpecs(specs []spec) []ui.Spec {
	converted := make([]ui.Spec, 0, len(specs))
	for _, item := range specs {
		converted = append(converted, ui.Spec{Key: item.Key, Name: item.Name, Value: item.Value, Score: item.Score})
	}
	return converted
}
