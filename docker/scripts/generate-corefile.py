#!/usr/bin/env python3

import os
import sys
from jinja2 import Template


def generate_corefile():
    # Read template
    template_path = "/templates/Corefile.j2"
    with open(template_path, 'r') as f:
        template_content = f.read()

    # Create template
    template = Template(template_content)

    # Get environment variables
    domain = os.getenv('TS_DOMAIN', '')
    hosts_file = os.getenv('TS_HOSTS_FILE', '')
    forward_to = os.getenv('TS_FORWARD_TO', '/etc/resolv.conf')
    additional_config = os.getenv('TS_ADDITIONAL_CONFIG', '')

    # If hosts_file is not explicitly set, check for the default location
    if not hosts_file or hosts_file.strip() == '':
        default_hosts_file = '/etc/ts-dns/hosts/custom_hosts'
        if os.path.exists(default_hosts_file):
            hosts_file = default_hosts_file
        else:
            hosts_file = None

    # Only use hosts_file if it's not empty
    if not hosts_file or hosts_file.strip() == '':
        hosts_file = None

    # Render template
    corefile_content = template.render(
        domain=domain,
        hosts_file=hosts_file,
        forward_to=forward_to
    )

    # Append additional configuration if provided
    if additional_config and additional_config.strip():
        corefile_content += "\n" + additional_config

    # Write to /Corefile
    with open('/Corefile', 'w') as f:
        f.write(corefile_content)

    print("Corefile generated:")
    print(corefile_content)


if __name__ == "__main__":
    generate_corefile()
