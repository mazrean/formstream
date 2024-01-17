import sys
import matplotlib.pyplot as plt


def parse_group(group: str):
    group = group.removeprefix("Benchmark")

    if group.startswith("FormStream"):
        group = group.removeprefix("FormStream")
        return f"FormStream({group})"
    elif group.startswith("StdMultipart_"):
        group = group.split("_")[1]
        return f"std(with {group})"

    return group


def parse_file_size(mem_size: str):
    if mem_size.endswith("MB"):
        return int(mem_size[:-2])
    elif mem_size.endswith("GB"):
        return int(mem_size[:-2]) * 2**10
    else:
        raise Exception("Unknown memory size: " + mem_size)


results = {}
for line in sys.stdin:
    if not line.startswith("Benchmark"):
        continue

    tokens = line.split()

    name_list = tokens[0].split("/")

    group = parse_group(name_list[0])

    results.setdefault(
        group,
        {
            "file_size": [],
            "time_per_ops": [],
            "mem_per_ops": [],
        },
    )

    str_file_size = name_list[1].split("-")[0]
    file_size = parse_file_size(str_file_size)
    results[group]["file_size"].append(file_size)

    time_per_ops = int(tokens[2])
    results[group]["time_per_ops"].append(time_per_ops)

    mem_per_ops = int(tokens[4])
    results[group]["mem_per_ops"].append(mem_per_ops)

time_per_ops_plt = plt.subplot(2, 1, 1)
mem_per_ops_plt = plt.subplot(2, 1, 2)
for group, result in results.items():
    time_per_ops_plt.plot(result["file_size"], result["time_per_ops"], label=group)
    mem_per_ops_plt.plot(result["file_size"], result["mem_per_ops"], label=group)

time_per_ops_plt.set_title("Time per operation")
time_per_ops_plt.set_xlabel("File size (MB)")
time_per_ops_plt.set_ylabel("Time per operation (ns)")
time_per_ops_plt.legend()

mem_per_ops_plt.set_title("Memory per operation")
mem_per_ops_plt.set_xlabel("File size (MB)")
mem_per_ops_plt.set_ylabel("Memory per operation (B)")
mem_per_ops_plt.legend()

plt.show()
