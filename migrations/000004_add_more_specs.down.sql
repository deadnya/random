DELETE FROM spec_odds WHERE spec_key IN (
    'even_number', 'odd_number', 'starts_with_1', 'ends_with_5', 'contains_7',
    'digit_sum_under_15', 'digit_sum_over_30', 'double_digit',
    'fibonacci', 'perfect_square', 'armstrong', 'ascending_digits', 'descending_digits',
    'contains_123', 'contains_456', 'digit_product_24', 'alternating_parity', 'three_consecutive'
);