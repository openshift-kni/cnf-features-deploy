#!/usr/bin/python

import os
import shutil
import shlex
import sys
import json
import yaml
from jinja2 import Template
import tempfile
import subprocess
from kubernetes import client, config
import logging


class Logger():
    @property
    def logger(self):
        fmt = '%(name)s %(asctime)s [%(levelname)s] \
            [%(module)s:%(lineno)s]: %(message)s'
        name = 'ztp-site-generator.watcher'
        lg = logging.getLogger(name)
        lg.setLevel(logging.DEBUG)
        formatter = logging.Formatter(
            fmt,
            datefmt='%Y-%m-%d %H:%M:%S %Z')

        if not lg.hasHandlers():
            # logging to console
            handler = logging.StreamHandler()
            handler.setLevel(logging.DEBUG)
            handler.setFormatter(formatter)
            lg.addHandler(handler)
        return lg


class SiteApi(Logger):
    def __init__(self):
        try:
            self.api = client.CustomObjectsApi()
            self.group = "ran.openshift.io"
            self.version = "v1"
            self.plural = "siteconfigs"
            self.watch = True
        except Exception as e:
            self.logger.exception(e)

    def watch_sites(self, rv):
        try:
            return self.api.list_cluster_custom_object_with_http_info(
                group=self.group, version=self.version,
                plural=self.plural, watch=self.watch,
                resource_version=rv, timeout_seconds=5)
        except Exception as e:
            self.logger.exception(e)


class PolicyGenWrapper(Logger):
    def __init__(self, paths: list):
        try:
            folders = [{'input': paths[0], 'output': paths[1]}]
            src = '/usr/src/hook/cnf-features-deploy'
            dest = '/tmp/cnf-features-deploy'
            shutil.copytree(src, dest)
            cwd = '/tmp/cnf-features-deploy/ztp/ztp-policy-generator'
            command = 'kustomize build --enable-alpha-plugins'
            oneliner_file = 'policyGenerator.yaml'
            env = os.environ.copy()
            env['XDG_CONFIG_HOME'] = cwd
            args = shlex.split(command)
            # Render policyGenerator.yaml template into cwd
            with open('pol_gen.yaml.j2', 'r') as tf:
                t = tf.read()
            tm = Template(t)
            pgy = tm.render(folders=folders)
            with open(os.path.join(cwd, oneliner_file), 'w') as of:
                of.write(pgy)
            self.logger.debug(f"Success writing {cwd}/{oneliner_file}: {pgy}")

            # Run policy generator
            with subprocess.Popen(
                            args, stderr=subprocess.PIPE,
                            stdout=subprocess.PIPE,
                            cwd=cwd, env=env) as pg:
                output = pg.communicate()
                if len(output[1]):
                    raise Exception(f"Manifest conversion failed: {output[1].decode()}")
        except Exception as e:
            self.logger.exception(f"PolicyGenWrapper failed: {e}")
            exit(1)


class OcWrapper(Logger):
    def __init__(self, action: str, path: str):
        try:
            status = None
            for f in self._find_files(path):
                cmd = ["oc", f"{action}", "-f", f"{f}"]
                self.logger.debug(cmd)
                status = subprocess.run(
                    cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    check=True)
                self.logger.debug(status.stdout.decode())
        except subprocess.CalledProcessError as cpe:
            nl = '\n'
            msg = f"{cpe.stdout.decode()} {cpe.stderr.decode()}"
            with open(f, 'r') as ef:
                err_file = ef.read()
            self.logger.debug(f"OC wrapper error:{nl}{err_file}")
            self.logger.exception(msg)
            raise Exception(f"Failed to {action} target manifests")
        except Exception as e:
            self.logger.exception(e)
            exit(1)

    def _find_files(self, root):
        for d, dirs, files in os.walk(root):
            for f in files:
                yield os.path.join(d, f)


class SiteResponseParser(Logger):
    def __init__(self, api_response, debug=False):
        if api_response[1] != 200:
            raise Exception(f"Site API call error: {api_response}")
        else:
            try:
                # Create temporary file structure for changed site manifests
                self.tmpdir = tempfile.mkdtemp()
                self.del_path = os.path.join(self.tmpdir, 'delete')
                self.del_list = []
                self.upd_path = os.path.join(self.tmpdir, 'update')
                self.upd_list = []
                os.mkdir(self.del_path)
                os.mkdir(self.upd_path)
                self._parse(api_response[0])
                self.logger.debug(f"Sites to delete are: {self.del_list}")
                self.logger.debug(
                    f"Sites to create/update are: {self.upd_list}")

                out_tmpdir = tempfile.mkdtemp()
                out_del_path = os.path.join(out_tmpdir, 'delete')
                out_upd_path = os.path.join(out_tmpdir, 'update')
                os.mkdir(out_del_path)
                os.mkdir(out_upd_path)
                # Do deletes
                if len(self.del_list) > 0:
                    PolicyGenWrapper([self.del_path, out_del_path])
                    OcWrapper('delete', out_del_path)
                else:
                    self.logger.debug("No objects to delete")

                # Do creates / updates
                if len(self.upd_list) > 0:
                    PolicyGenWrapper([self.upd_path, out_upd_path])
                    OcWrapper('apply', out_upd_path)
                else:
                    self.logger.debug("No objects to update")

            except Exception as e:
                self.logger.exception(f"Exception by SiteResponseParser: {e}")
                exit(1)
            finally:
                if not debug:
                    shutil.rmtree(self.tmpdir)
                    shutil.rmtree(out_tmpdir)

    def _parse(self, resp_data):
        # The response comes in two flavors:
        # 1. For a single object - as a dictionary
        # 2. For several objects - as a text, that must be split to a list
        try:
            if type(resp_data) == str and len(resp_data):
                resp_list = resp_data.split('\n')
                items = (x for x in resp_list if len(x) > 0)
                for item in items:
                    self._create_site_file(json.loads(item))
            elif type(resp_data) == dict:
                self._create_site_file(resp_data)
            else:
                pass  # Empty response - no changes
        except Exception as e:
            self.logger.Exception(
                f"Exception when parsing API response: {e}")

    def _prune_managed_info(self, site: dict):
        site['object']['metadata'].pop("annotations", None)
        site['object']['metadata'].pop("creationTimestamp", None)
        site['object']['metadata'].pop("managedFields", None)
        site['object']['metadata'].pop("generation", None)
        site['object']['metadata'].pop("resourceVersion", None)
        site['object']['metadata'].pop("selfLink", None)
        site['object']['metadata'].pop("uid", None)

    def _create_site_file(self, site: dict):
        try:
            self._prune_managed_info(site)
            action = site.get("type")
            if action == "DELETED":
                path, lst = self.del_path, self.del_list
            else:
                path, lst = self.upd_path, self.upd_list
            handle, name = tempfile.mkstemp(dir=path)
            with open(name, 'w') as f:
                yaml.dump(site.get("object"), f)
            lst.append(site.get("object").get("metadata").get("name"))
        except Exception as e:
            self.logger.exception(e)
            exit(1)


if __name__ == '__main__':
    try:
        lg = Logger()
        config.load_incluster_config()
        site_api = SiteApi()
        resp = site_api.watch_sites(sys.argv[1])
        debug = len(sys.argv) > 2
        SiteResponseParser(resp, debug=debug)
    except Exception as e:
        lg.logger.exception(e)
