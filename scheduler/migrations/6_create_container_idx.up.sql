create index containers_scheduling_idx ON containers (status, cpu_class_min, cpu_class_max, disk_class_min,
                                                      disk_class_max, gpu_class_min, gpu_class_max);