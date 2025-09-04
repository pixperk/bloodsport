def two_sum(nums, target):
    seen = {}  # value -> index
    for i, x in enumerate(nums):
        need = target - x
        if need in seen:
            return (seen[need], i)
        seen[x] = i
    return None  # if no solution found

# Read input
line1 = input().split(',')
nums = [int(x) for x in line1]
target = int(input())
result = two_sum(nums, target)

# Print the result in the expected format
if result:
    print(f"{result[0]},{result[1]}")
else:
    print("No solution found")
