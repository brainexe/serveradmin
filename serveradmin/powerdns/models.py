from django.db import models

# We provide the PowerDNS database schema for PostgreSQL with a Django
# migration using RunSQL. It creates the necessary relations etc. if they
# do not already exist.
#
# But we DON'T want Django to manage the based on the models because below
# models only contain the fields we need for the "Serveradmin to PowerDNS"
# integration and therefore are incomplete.


class Domain(models.Model):
    """PowerDNS Domain

    Model to access domain in the PowerDNS db.

    See https://doc.powerdns.com/authoritative/backends/generic-postgresql.html#default-schema
    """
    TYPES = [
        ('MASTER', 'MASTER'),
        ('SLAVE', 'SLAVE'),
        ('NATIVE', 'NATIVE'),
    ]

    id = models.IntegerField(primary_key=True)
    name = models.CharField(max_length=255, null=False)
    type = models.CharField(max_length=6, null=False, choices=TYPES)

    class Meta:
        managed = False
        db_table = 'domains'


class Record(models.Model):
    """PowerDNS Record

    Model to access records in the PowerDNS db.

    See https://doc.powerdns.com/authoritative/backends/generic-postgresql.html#default-schema
    """
    TYPES = [
        ('A', 'A'),
        ('AAAA', 'AAAA'),
        ('CNAME', 'CNAME'),
        ('TXT', 'TXT'),
        ('SSHFP', 'SSHFP'),
        ('SOA', 'SOA'),
        ('MX', 'MX'),
        ('PTR', 'PTR'),
        ('NS', 'NS'),
    ]

    id = models.BigIntegerField(primary_key=True)
    domain = models.ForeignKey('Domain', on_delete=models.CASCADE)
    name = models.CharField(max_length=255, default=None)
    type = models.CharField(max_length=10, default=None, choices=TYPES)
    content = models.CharField(max_length=65535, default=None)
    ttl = models.IntegerField()

    class Meta:
        managed = False
        db_table = 'records'
