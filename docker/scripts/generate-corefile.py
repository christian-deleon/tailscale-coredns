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

    # Only use hosts_file if it's not empty
    if not hosts_file or hosts_file.strip() == '':
        hosts_file = None

    # Render template
    corefile_content = template.render(
        domain=domain,
        hosts_file=hosts_file,
        forward_to=forward_to
    )

    # Write to /Corefile
    with open('/Corefile', 'w') as f:
        f.write(corefile_content)

    print("Corefile generated:")
    print(corefile_content)


if __name__ == "__main__":
    generate_corefile()
