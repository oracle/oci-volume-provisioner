#!/usr/bin/env python

# Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import time
import sys
import select
import subprocess
from shutil import copyfile

DEBUG_FILE = "runner.log"
REPORT_DIR_PATH="/tmp/results"
REPORT_FILE="done"
TMP_KUBECONFIG = "/tmp/kubeconfig.conf"

def _banner(as_banner, bold):
    if as_banner:
        if bold:
            print "********************************************************"
        else:
            print "--------------------------------------------------------"

def _process_stream(stream, read_fds, global_buf, line_buf):
    char = stream.read(1)
    if char == '':
        read_fds.remove(stream)
    global_buf.append(char)
    line_buf.append(char)
    if char == '\n':
        _debug_file(''.join(line_buf))
        line_buf = []
    return line_buf

def _poll(stdout, stderr):
    stdoutbuf = []
    stdoutbuf_line = []
    stderrbuf = []
    stderrbuf_line = []
    read_fds = [stdout, stderr]
    x_fds = [stdout, stderr]
    while read_fds:
        rlist, _, _ = select.select(read_fds, [], x_fds)
        if rlist:
            for stream in rlist:
                if stream == stdout:
                    stdoutbuf_line = _process_stream(stream, read_fds, stdoutbuf, stdoutbuf_line)
                if stream == stderr:
                    stderrbuf_line = _process_stream(stream, read_fds, stderrbuf, stderrbuf_line)
    return (''.join(stdoutbuf), ''.join(stderrbuf))

# On exit return 0 for success or any other integer for a failure.
# If write_report is true then write a completion file to the Sonabuoy plugin result file.
# The default location is: /tmp/results/done
def finish_with_exit_code(exit_code, write_report=True, report_dir_path=REPORT_DIR_PATH, report_file=REPORT_FILE):
    print "finishing with exit code: " + str(exit_code)
    if write_report:
        if not os.path.exists(report_dir_path):
            os.makedirs(report_dir_path)
        if exit_code == 0:
            _debug_file("\nTest Suite Success\n")
        else:
            _debug_file("\nTest Suite Failed\n")
        time.sleep(3)
        copyfile(DEBUG_FILE, report_dir_path + "/" + DEBUG_FILE)
        with open(report_dir_path + "/" + report_file, "w+") as file:
            file.write(str(report_dir_path + "/" + DEBUG_FILE))
    sys.exit(exit_code)

def reset_debug_file():
    if os.path.exists(DEBUG_FILE):
        os.remove(DEBUG_FILE)

def _debug_file(string):
    with open(DEBUG_FILE, "a") as debug_file:
        debug_file.write(string)


def log(string, as_banner=False, bold=False):
    _banner(as_banner, bold)
    print string
    _banner(as_banner, bold)

def run_command(cmd, cwd, display_errors=True):
    log(cwd + ": " + cmd)
    process = subprocess.Popen(cmd,
                               stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE,
                               shell=True, cwd=cwd)
    (stdout, stderr) = _poll(process.stdout, process.stderr)
    returncode = process.wait()
    if returncode != 0 and display_errors:
        log("    stdout: " + stdout)
        log("    stderr: " + stderr)
        log("    result: " + str(returncode))
    return (stdout, stderr, returncode)

def _get_kubeconfig():
    return os.environ['KUBECONFIG'] if "KUBECONFIG" in os.environ else TMP_KUBECONFIG

def kubectl(action, exit_on_error=True, display_errors=True, log_stdout=True):
    if "KUBECONFIG" not in os.environ and "KUBECONFIG_VAR" not in os.environ:
        (stdout, _, returncode) = run_command("kubectl " + action, ".", display_errors)
    else:
        (stdout, _, returncode) = run_command("KUBECONFIG=" + _get_kubeconfig() + " kubectl " + action, ".", display_errors)
    if exit_on_error and returncode != 0:
        log("Error running kubectl")
        finish_with_exit_code(1)
    if log_stdout:
        log(stdout)
    return stdout
