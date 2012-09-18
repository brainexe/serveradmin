try:
    import simplejson as json
except ImportError:
    import json

from django.http import (HttpResponse, HttpResponseBadRequest, 
        HttpResponseRedirect, Http404)
from django.template.response import TemplateResponse
from django.contrib.auth.decorators import login_required, permission_required
from django.views.decorators.csrf import ensure_csrf_cookie
from django.core.urlresolvers import reverse
from django.contrib import messages
from django import forms

from adminapi.utils import IP
from adminapi.utils.json import json_encode_extra
from adminapi.utils.parse import parse_query
from serveradmin.dataset import query, filters, DatasetError
from serveradmin.dataset.filters import filter_classes
from serveradmin.dataset.base import lookups
from serveradmin.dataset.commit import commit_changes, CommitValidationFailed
from serveradmin.dataset.values import get_attribute_values
from serveradmin.dataset.typecast import typecast
from serveradmin.dataset.models import ServerType
from serveradmin.dataset.create import create_server

MAX_DISTINGUISHED_VALUES = 50
NUM_SERVERS_DEFAULT = 25

@login_required
@ensure_csrf_cookie
def index(request):
    return TemplateResponse(request, 'servershell/index.html', {
        'attribute_list': sorted(lookups.attr_names.keys()),
        'search_term': request.GET.get('term', '')
    })

@login_required
def autocomplete(request):
    autocomplete_list = []
    if 'hostname' in request.GET:
        hostname = request.GET['hostname']
        try:
            hosts = query(hostname=filters.Startswith(hostname)).limit(10)
            autocomplete_list += (host['hostname'] for host in hosts)
        except DatasetError:
            pass # If there is no valid query, just don't autocomplete

    return HttpResponse(json.dumps({'autocomplete': autocomplete_list}),
            mimetype='application/x-json')

@login_required
def get_results(request):
    term = request.GET.get('term', '')
    try:
        offset = int(request.GET.get('offset', '0'))
        limit = min(int(request.GET.get('limit', '0')), 250)
    except ValueError:
        offset = 0
        limit = NUM_SERVERS_DEFAULT

    order_by = request.GET.get('order_by')
    order_dir = request.GET.get('order_dir', 'asc')
    
    shown_attributes = ['hostname', 'intern_ip', 'servertype']
    try:
        query_args = parse_query(term, filter_classes)
        
        # Add attributes with non-constant values and multi attributes
        # to the shown attributes
        for attr, value in query_args.iteritems():
            try:
                multi = lookups.attr_names[attr].multi
            except KeyError:
                continue
            if not isinstance(value, (filters.ExactMatch, basestring)) or multi:
                # FIXME: Just a dirty workaround
                if attr == 'all_ips':
                    if u'intern_ip' not in shown_attributes:
                        shown_attributes.append(u'additional_ips')
                    if u'additional_ips' not in shown_attributes:
                        shown_attributes.append(u'additional_ips')
                    continue
                if attr not in shown_attributes:
                    shown_attributes.append(attr)
        
        q = query(**query_args).limit(offset, limit)
        if order_by:
            q = q.order_by(order_by, order_dir)
        results = q.get_raw_results()
        num_servers = q.get_num_rows()
    except (ValueError, DatasetError), e:
        return HttpResponse(json.dumps({
            'status': 'error',
            'message': e.message
        }))

    return HttpResponse(json.dumps({
        'status': 'success',
        'understood': q.get_representation().as_code(hide_extra=True),
        'servers': results,
        'num_servers': num_servers,
        'shown_attributes': shown_attributes,
    }, default=json_encode_extra), mimetype='application/x-json')

@login_required
def export(request):
    term = request.GET.get('term', '')
    try:
        query_args = parse_query(term, filter_classes)
        q = query(**query_args).restrict('hostname')
    except (ValueError, DatasetError), e:
        return HttpResponse(e.message, status=400)

    hostnames = u' '.join(server['hostname'] for server in q)
    return HttpResponse(hostnames, mimetype='text/plain')

def list_and_edit(request, mode='list'):
    try:
        object_id = request.GET['object_id']
        server = query(object_id=object_id).get()
    except (KeyError, DatasetError):
        raise Http404

    if not request.user.has_perm('dataset.change_serverobject'):
        mode = 'list'

    stype = lookups.stype_names[server['servertype']]
    non_editable = ['servertype']
    
    invalid_attrs = set()
    if mode == 'edit' and request.POST:
        attrs = set(request.POST.getlist('attr'))
        for attr in attrs:
            if attr in non_editable:
                continue
            if lookups.attr_names[attr].multi:
                lines = request.POST.get('attr_' + attr, '').splitlines()
                value = set()
                for line in lines:
                    value.add(typecast(attr, line.strip()))
            else:
                value = typecast(attr, request.POST.get('attr_' + attr, ''))
            server[attr] = value
        for attr in server.keys():
            if attr in non_editable:
                continue
            if attr not in attrs:
                del server[attr]
        try:
            server.commit()
            messages.success(request, 'Edited server successfully')
            url = '{0}?object_id={1}'.format(reverse('servershell_list'),
                    server.object_id)
            return HttpResponseRedirect(url)
        except CommitValidationFailed as e:
            invalid_attrs = set([attr for obj_id, attr in e.violations])

    fields = []
    fields_set = set()
    for key, value in server.iteritems():
        fields_set.add(key)
        fields.append({
            'key': key,
            'value': value,
            'has_value': True,
            'editable': key not in non_editable,
            'type': lookups.attr_names[key].type,
            'multi': lookups.attr_names[key].multi,
            'required': lookups.stype_attrs[(stype.name, key)].required,
            'error': key in invalid_attrs
        })
    
    if mode == 'edit':
        for attr in stype.attributes:
            if attr.name in fields_set:
                continue
            fields.append({
                'key': attr.name,
                'value': [] if attr.multi else '',
                'has_value': False,
                'editable': True,
                'type': attr.type,
                'multi': attr.multi,
                'required': False,
                'error': attr.name in invalid_attrs
            })
    
    # Sort keys by some order and then lexographically
    _key_order = ['hostname', 'servertype', 'intern_ip']
    _key_order_lookup = dict((key, i) for i, key in enumerate(_key_order))
    def _sort_key(x):
        return (_key_order_lookup.get(x['key'], 100), x['key'])
    fields.sort(key=_sort_key)
    
    return TemplateResponse(request, 'servershell/{0}.html'.format(mode), {
        'object_id': server.object_id,
        'fields': fields,
        'is_ajax': request.is_ajax(),
        'base_template': 'empty.html' if request.is_ajax() else 'base.html',
        'link': request.get_full_path()
    })

@login_required
@permission_required('dataset.change_serverobject')
def commit(request):
    try:
        commit = json.loads(request.POST['commit'])
    except (KeyError, ValueError):
        return HttpResponseBadRequest()

    if 'changes' in commit:
        changes = {}
        for key, value in commit['changes'].iteritems():
            if not key.isdigit():
                continue
            changes[int(key)] = value
        commit['changes'] = changes

    try:
        commit_changes(commit)
    except (ValueError, DatasetError) as e:
        return HttpResponseBadRequest()
    except CommitValidationFailed, e:
        result = {
            'status': 'error',
            'message': e.message
        }
    else:
        result = {
            'status': 'success'
        }

    return HttpResponse(json.dumps(result), mimetype='application/x-json')

@login_required
def get_values(request):
    try:
        attr_obj = lookups.attr_names[request.GET['attribute']]
    except KeyError:
        raise Http404

    values = get_attribute_values(attr_obj.name, MAX_DISTINGUISHED_VALUES)

    return TemplateResponse(request, 'servershell/values.html', {
        'attribute': attr_obj,
        'values': values,
        'num_values': MAX_DISTINGUISHED_VALUES
    })

@login_required
@permission_required('dataset.create_serverobject')
def new_server(request):
    class NewServerForm(forms.Form):
        hostname = forms.CharField()
        intern_ip = forms.IPAddressField()
        segment = forms.CharField()
        servertype = forms.ModelChoiceField(queryset=ServerType.objects.order_by(
            'name'))

    if request.method == 'POST':
        form = NewServerForm(request.POST)
        if form.is_valid():
            attributes = form.cleaned_data.copy()
            attributes['intern_ip'] = IP(attributes['intern_ip'])
            attributes['servertype'] = attributes['servertype'].name
            server_id = create_server(attributes, skip_validation=True,
                    fill_defaults=True, fill_defaults_all=True)
            url = '{0}?object_id={1}'.format(reverse('servershell_edit'),
                    server_id)
            return HttpResponseRedirect(url)
    else:
        form = NewServerForm()

    return TemplateResponse(request, 'servershell/new_server.html', {
        'form': form,
        'is_ajax': request.is_ajax(),
        'base_template': 'empty.html' if request.is_ajax() else 'base.html'
    })
