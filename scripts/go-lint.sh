#!/bin/sh
yamllint --no-warnings -c scripts/yamllint.config deploy/crs
dirs=(cmd pkg)
for dir in "${dirs[@]}"
do
    if ! golint -set_exit_status ${dir}/...; then
        code=1
    fi
done
exit ${code:0}
