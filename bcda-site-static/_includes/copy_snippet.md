{% assign code = include.code %}

<div class="highlight"><pre class="highlight"><code>{{code}}
</code></pre></div>

{% assign nanosecond = "now" | date: "%N" %}
<input type="text" id="code{{ nanosecond }}" style="position: absolute;left: -1000px;" value="{{ code }}"/>
<div class="button" on>
    <p>
        <a href="javascript:void(0)" onclick="copyText{{ nanosecond }}()" id="copybutton{{ nanosecond }}">
          Copy to clipboard
        </a>
    </p>
</div>

<script>
function copyText{{ nanosecond }}(){
  /* Get the text field */
  var copyText = document.getElementById("code{{ nanosecond }}");

  /* Select the text field */
  copyText.select();

  /* Copy the text inside the text field */
  document.execCommand("copy");
}
</script>
