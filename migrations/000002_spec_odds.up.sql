CREATE TABLE IF NOT EXISTS spec_odds (
    spec_key TEXT PRIMARY KEY,
    probability DOUBLE PRECISION NOT NULL CHECK (probability > 0 AND probability <= 1),
    description TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO spec_odds (spec_key, probability, description)
VALUES
    ('prime', 0.078498, 'Only divisible by 1 and itself'),
    ('all_digits_equal', 0.00001, 'All digits identical'),
    ('palindrome', 0.001, 'Reads the same forwards and backwards'),
    ('unique_digits_6', 0.1512, 'All six digits unique'),
    ('unique_digits_5', 0.4536, 'Exactly five unique digits'),
    ('unique_digits_4', 0.3276, 'Exactly four unique digits'),
    ('divisible_by_9', 0.111112, 'Digit sum divisible by 9'),
    ('divisible_by_7', 0.142858, 'Divisible by 7'),
    ('trailing_000', 0.001, 'Number ends with 000'),
    ('exact_zero', 0.000001, 'Rolled number is exactly 000000'),
    ('even_number', 0.5, 'Divisible by 2'),
    ('odd_number', 0.5, 'Not divisible by 2'),
    ('starts_with_1', 0.1, 'Begins with the loneliest number'),
    ('ends_with_5', 0.1, 'Ends with a hand'),
    ('contains_7', 0.488, 'Contains the lucky digit'),
    ('digit_sum_under_15', 0.25, 'Sum under 15'),
    ('digit_sum_over_30', 0.25, 'Sum over 30'),
    ('double_digit', 0.488, 'Has consecutive identical digits'),
    ('fibonacci', 0.008, 'Number is a Fibonacci number'),
    ('perfect_square', 0.001, 'Number is a perfect square'),
    ('armstrong', 0.0005, 'Number is an Armstrong number'),
    ('ascending_digits', 0.0001, 'Digits are in strictly ascending order'),
    ('descending_digits', 0.0001, 'Digits are in strictly descending order'),
    ('contains_123', 0.004, 'Number contains the sequence 123'),
    ('contains_456', 0.004, 'Number contains the sequence 456'),
    ('digit_product_24', 0.01, 'Product of digits equals 24'),
    ('alternating_parity', 0.125, 'Digits alternate between odd and even'),
    ('three_consecutive', 0.015, 'Number contains three consecutive digits')
ON CONFLICT (spec_key) DO UPDATE
SET
    probability = EXCLUDED.probability,
    description = EXCLUDED.description,
    updated_at = NOW();
