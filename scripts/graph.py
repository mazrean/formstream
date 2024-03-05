import sys
import os.path as path
import matplotlib.pyplot as plt
import numpy as np


def parse_group(group: str):
    group = group.removeprefix("Benchmark")

    if group.startswith("FormStream"):
        group = group.removeprefix("FormStream")
        return f"FormStream({group})"
    elif group.startswith("StdMultipart"):
        group = group.removeprefix("StdMultipart")
        return f"std(with {group})"

    return group


def parse_file_size(mem_size: str):
    if mem_size.endswith("MB"):
        return int(mem_size[:-2])
    elif mem_size.endswith("GB"):
        return int(mem_size[:-2]) * 2**10
    else:
        raise Exception("Unknown memory size: " + mem_size)


file_sizes = []
time_per_ops = {}
mem_per_ops = {}
for line in sys.stdin:
    if not line.startswith("Benchmark"):
        continue

    tokens = line.split()

    name_list = tokens[0].split("/")

    group = parse_group(name_list[0])

    file_size = name_list[1].split("-")[0]
    file_sizes.append(file_size)

    time_per_ops.setdefault(group, {})
    time_per_ops[group][file_size] = int(tokens[2])

    mem_per_ops.setdefault(group, {})
    mem_per_ops[group][file_size] = int(tokens[4])

# remove duplicates
file_sizes = sorted(set(file_sizes), key=file_sizes.index)

bar_width = 0.25

index = np.arange(len(file_sizes))
fig, ax_time = plt.subplots(figsize=(12, 7))
for i, (group, group_time_dict) in enumerate(time_per_ops.items()):
    ax_time.bar(
        index + i * bar_width,
        [group_time_dict[fs] / 1e6 for fs in file_sizes],
        bar_width,
        label=group,
    )

ax_time.set_xlabel("File Size")
ax_time.set_ylabel("Execution Time (ms)")
ax_time.set_title("Execution Time Comparison (Log Scale)")
ax_time.set_xticks(index + bar_width)
ax_time.set_xticklabels(file_sizes)
ax_time.set_yscale("log")
ax_time.legend()

fig.savefig(path.join(path.dirname(__file__), "../docs/images/time.png"))

fig, ax_memory = plt.subplots(figsize=(12, 7))
for i, (group, group_mem_dict) in enumerate(mem_per_ops.items()):
    ax_memory.bar(
        index + i * bar_width,
        [group_mem_dict[fs] for fs in file_sizes],
        bar_width,
        label=group,
    )

ax_memory.set_xlabel("File Size")
ax_memory.set_ylabel("Memory Usage (Bytes)")
ax_memory.set_title("Memory Usage Comparison (Log Scale)")
ax_memory.set_xticks(index + bar_width)
ax_memory.set_xticklabels(file_sizes)
ax_memory.set_yscale("log")
ax_memory.legend()

fig.savefig(path.join(path.dirname(__file__), "../docs/images/memory.png"))
